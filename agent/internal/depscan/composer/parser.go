package composer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"devsecops-agent/internal/depscan"
)

type auditReport struct {
	Advisories map[string][]advisory `json:"advisories"`
}

type advisory struct {
	AdvisoryID       string        `json:"advisoryId"`
	PackageName      string        `json:"packageName"`
	CVE              string        `json:"cve"`
	Title            string        `json:"title"`
	Description      string        `json:"description"`
	Link             string        `json:"link"`
	Severity         any           `json:"severity"`
	AffectedVersions any           `json:"affectedVersions"`
	ReportedAt       string        `json:"reportedAt"`
	Sources          []sourceEntry `json:"sources"`
}

type sourceEntry struct {
	Name     string `json:"name"`
	RemoteID string `json:"remoteId"`
}

// Composer versions may return an empty advisories array
// when no vulnerabilities are found, or an object grouped
// by package name when advisories exist.
func (r *auditReport) UnmarshalJSON(data []byte) error {
	var raw struct {
		Advisories json.RawMessage `json:"advisories"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Advisories = make(map[string][]advisory)

	advisories := bytes.TrimSpace(raw.Advisories)

	if len(advisories) == 0 ||
		bytes.Equal(advisories, []byte("null")) {
		return nil
	}

	switch advisories[0] {
	case '{':
		return json.Unmarshal(advisories, &r.Advisories)

	case '[':
		var items []advisory

		if err := json.Unmarshal(advisories, &items); err != nil {
			return err
		}

		for _, item := range items {
			if item.PackageName == "" {
				continue
			}

			r.Advisories[item.PackageName] = append(
				r.Advisories[item.PackageName],
				item,
			)
		}

		return nil

	default:
		return fmt.Errorf(
			"unexpected Composer advisories format: %s",
			advisories,
		)
	}
}

type lockFile struct {
	Packages []lockedPackage `json:"packages"`

	PackagesDev []lockedPackage `json:"packages-dev"`
}

type lockedPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func loadVersionsFromContent(content []byte) map[string]string {
	versions := make(map[string]string)

	var lock lockFile

	if err := json.Unmarshal(content, &lock); err != nil {
		return versions
	}

	for _, pkg := range lock.Packages {
		versions[pkg.Name] = pkg.Version
	}

	for _, pkg := range lock.PackagesDev {
		versions[pkg.Name] = pkg.Version
	}

	return versions
}

// Kept for existing parser tests.
func loadVersions(lockPath string) map[string]string {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return make(map[string]string)
	}

	return loadVersionsFromContent(content)
}

func jsonValueString(value any) string {
	if value == nil {
		return ""
	}

	if text, ok := value.(string); ok {
		return text
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}

	return string(encoded)
}

func findingID(item advisory, packageName, affected string) string {
	if item.AdvisoryID != "" {
		return item.AdvisoryID
	}

	if item.CVE != "" {
		return item.CVE
	}

	for _, source := range item.Sources {
		if source.RemoteID != "" {
			return source.RemoteID
		}
	}

	return fmt.Sprintf(
		"composer:%s:%s:%s",
		packageName,
		affected,
		item.Title,
	)
}

func parseFindings(
	report auditReport,
	lockPath string,
	manifest string,
) []depscan.Finding {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return nil
	}

	return parseFindingsFromContent(
		report,
		content,
		manifest,
	)
}

func parseFindingsFromContent(
	report auditReport,
	lockContent []byte,
	manifest string,
) []depscan.Finding {
	versions := loadVersionsFromContent(lockContent)
	findings := make([]depscan.Finding, 0)

	packages := make([]string, 0, len(report.Advisories))

	for packageName := range report.Advisories {
		packages = append(packages, packageName)
	}

	sort.Strings(packages)

	for _, packageName := range packages {
		for _, item := range report.Advisories[packageName] {
			name := item.PackageName

			if name == "" {
				name = packageName
			}

			affected := jsonValueString(
				item.AffectedVersions,
			)

			title := item.Title

			if title == "" {
				title = "Security advisory affecting " + name
			}

			description := item.Description

			if description == "" {
				description = affected
			}

			metadata := map[string]any{
				"ecosystem":         "composer",
				"manifest_file":     manifest,
				"advisory_id":       item.AdvisoryID,
				"affected_versions": item.AffectedVersions,
				"reported_at":       item.ReportedAt,
				"sources":           item.Sources,
			}

			findings = append(findings, depscan.Finding{
				FindingType:      "vulnerability",
				FindingID:        findingID(item, name, affected),
				VulnerabilityID:  item.CVE,
				PackageName:      name,
				InstalledVersion: versions[name],
				Severity: depscan.NormalizeSeverity(
					jsonValueString(item.Severity),
				),
				Title:       title,
				Description: description,
				PrimaryURL:  item.Link,
				Metadata:    metadata,
			})
		}
	}

	return findings
}
