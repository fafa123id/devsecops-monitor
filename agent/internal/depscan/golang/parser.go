package golang

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"devsecops-agent/internal/depscan"
)

type govulnMessage struct {
	SBOM    *govulnSBOM    `json:"SBOM"`
	OSV     *goOSVEntry    `json:"osv"`
	Finding *govulnFinding `json:"finding"`
	Config  *govulnConfig  `json:"config"`
}

type govulnConfig struct {
	ScanLevel string `json:"scan_level"`
}

type govulnSBOM struct {
	Modules []*goModule `json:"modules"`
}

type goModule struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type goOSVEntry struct {
	ID               string         `json:"id"`
	Summary          string         `json:"summary"`
	Details          string         `json:"details"`
	Aliases          []string       `json:"aliases"`
	References       []goReference  `json:"references"`
	DatabaseSpecific map[string]any `json:"database_specific"`
}

type goReference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type govulnFinding struct {
	OSV          string         `json:"osv"`
	FixedVersion string         `json:"fixed_version"`
	Trace        []goTraceFrame `json:"trace"`
}

type goTraceFrame struct {
	Module   string      `json:"module"`
	Version  string      `json:"version"`
	Package  string      `json:"package"`
	Function string      `json:"function"`
	Receiver string      `json:"receiver"`
	Position *goPosition `json:"position"`
}

type goPosition struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

type goFindingAggregate struct {
	Finding    depscan.Finding
	TraceCount int
	Traces     []goTraceFrame
	TraceKeys  map[string]bool
}

func parseFindings(data []byte) ([]depscan.Finding, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	osvEntries := make(map[string]goOSVEntry)
	moduleVersions := make(map[string]string)
	findings := make([]govulnFinding, 0)

	scanLevel := ""

	messageCount := 0

	// govulncheck emits a stream of independent JSON messages.
	// Collect all messages first because their order is not guaranteed.
	for {
		var message govulnMessage

		err := decoder.Decode(&message)

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf(
				"decode govulncheck message: %w",
				err,
			)
		}

		messageCount++

		if message.Config != nil {
			scanLevel = message.Config.ScanLevel
		}

		if message.SBOM != nil {
			for _, module := range message.SBOM.Modules {
				if module == nil || module.Path == "" {
					continue
				}

				moduleVersions[module.Path] = module.Version
			}
		}

		if message.OSV != nil && message.OSV.ID != "" {
			osvEntries[message.OSV.ID] = *message.OSV
		}

		if message.Finding != nil && message.Finding.OSV != "" {
			findings = append(findings, *message.Finding)
		}
	}

	if messageCount == 0 {
		return nil, fmt.Errorf("govulncheck returned an empty JSON stream")
	}

	aggregates := make(map[string]*goFindingAggregate)

	for _, item := range findings {
		osvID := strings.TrimSpace(item.OSV)

		if osvID == "" {
			continue
		}

		entry := osvEntries[osvID]

		var frame goTraceFrame

		if len(item.Trace) > 0 {
			frame = item.Trace[0]
		}

		module := strings.TrimSpace(frame.Module)
		pkg := strings.TrimSpace(frame.Package)

		packageName := module

		if packageName == "" {
			packageName = pkg
		}

		if packageName == "" {
			packageName = "stdlib"
		}

		version := strings.TrimSpace(frame.Version)

		if version == "" && module != "" {
			version = moduleVersions[module]
		}

		location := packageName

		if pkg != "" && pkg != module {
			location += ":" + pkg
		}

		key := strings.Join([]string{
			osvID,
			module,
			pkg,
		}, "\x00")

		aggregate, exists := aggregates[key]

		if !exists {
			title := entry.Summary

			if title == "" {
				title = "Go security vulnerability " + osvID
			}

			metadata := map[string]any{
				"ecosystem":           "golang",
				"go_vulnerability_id": osvID,
				"aliases":             entry.Aliases,
				"module":              module,
				"go_package":          pkg,
				"function":            frame.Function,
				"receiver":            frame.Receiver,
				"location":            location,
				"scan_level":          scanLevel,
			}

			if frame.Position != nil {
				metadata["source_location"] = map[string]any{
					"filename": frame.Position.Filename,
					"line":     frame.Position.Line,
					"column":   frame.Position.Column,
				}
			}

			aggregate = &goFindingAggregate{
				Finding: depscan.Finding{
					FindingType:      "vulnerability",
					FindingID:        osvID,
					VulnerabilityID:  vulnerabilityID(entry, osvID),
					PackageName:      packageName,
					InstalledVersion: version,
					FixedVersion:     item.FixedVersion,
					Severity:         goSeverity(entry),
					Title:            title,
					Description:      entry.Details,
					PrimaryURL:       goPrimaryURL(entry, osvID),
					Metadata:         metadata,
				},
				Traces:    make([]goTraceFrame, 0),
				TraceKeys: make(map[string]bool),
			}

			aggregates[key] = aggregate
		}

		// Preserve all unique traces without creating duplicate
		// canonical findings for the same advisory and package.
		traceJSON, err := json.Marshal(item.Trace)
		if err != nil {
			return nil, fmt.Errorf("encode Go finding trace: %w", err)
		}

		traceKey := string(traceJSON)

		if !aggregate.TraceKeys[traceKey] {
			aggregate.TraceKeys[traceKey] = true
			aggregate.Traces = append(
				aggregate.Traces,
				item.Trace...,
			)
		}

		if len(item.Trace) > aggregate.TraceCount {
			aggregate.TraceCount = len(item.Trace)

			if frame.Function != "" {
				aggregate.Finding.Metadata["function"] = frame.Function
			}

			if frame.Receiver != "" {
				aggregate.Finding.Metadata["receiver"] = frame.Receiver
			}
		}

		if aggregate.Finding.InstalledVersion == "" && version != "" {
			aggregate.Finding.InstalledVersion = version
		}

		if aggregate.Finding.FixedVersion == "" {
			aggregate.Finding.FixedVersion = item.FixedVersion
		}
	}

	result := make([]depscan.Finding, 0, len(aggregates))

	for _, aggregate := range aggregates {
		aggregate.Finding.Metadata["traces"] = aggregate.Traces

		result = append(result, aggregate.Finding)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].PackageName != result[j].PackageName {
			return result[i].PackageName < result[j].PackageName
		}

		return result[i].FindingID < result[j].FindingID
	})

	return result, nil
}

func vulnerabilityID(entry goOSVEntry, fallback string) string {
	// Prefer CVE, then GHSA, then other aliases.
	for _, alias := range entry.Aliases {
		if strings.HasPrefix(strings.ToUpper(alias), "CVE-") {
			return alias
		}
	}

	for _, alias := range entry.Aliases {
		if strings.HasPrefix(strings.ToUpper(alias), "GHSA-") {
			return alias
		}
	}

	if len(entry.Aliases) > 0 {
		return entry.Aliases[0]
	}

	return fallback
}

func goSeverity(entry goOSVEntry) string {
	severity, ok := entry.DatabaseSpecific["severity"].(string)

	if !ok {
		return "unknown"
	}

	return depscan.NormalizeSeverity(severity)
}

func goPrimaryURL(entry goOSVEntry, fallback string) string {
	for _, reference := range entry.References {
		if strings.EqualFold(reference.Type, "WEB") &&
			strings.TrimSpace(reference.URL) != "" {
			return reference.URL
		}
	}

	return "https://pkg.go.dev/vuln/" + fallback
}
