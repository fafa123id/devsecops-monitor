package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"devsecops-agent/internal/depscan"
)

type auditReport struct {
	AuditReportVersion int                      `json:"auditReportVersion"`
	Vulnerabilities    map[string]vulnerability `json:"vulnerabilities"`
}

type vulnerability struct {
	Name         string            `json:"name"`
	Severity     string            `json:"severity"`
	IsDirect     bool              `json:"isDirect"`
	Via          []json.RawMessage `json:"via"`
	Effects      []string          `json:"effects"`
	Range        string            `json:"range"`
	Nodes        []string          `json:"nodes"`
	FixAvailable any               `json:"fixAvailable"`
}

type advisory struct {
	Source     int      `json:"source"`
	Name       string   `json:"name"`
	Dependency string   `json:"dependency"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	Range      string   `json:"range"`
	CWE        []string `json:"cwe"`

	CVSS struct {
		Score        float64 `json:"score"`
		VectorString string  `json:"vectorString"`
	} `json:"cvss"`
}

type lockFile struct {
	Packages     map[string]lockPackage     `json:"packages"`
	Dependencies map[string]json.RawMessage `json:"dependencies"`
}

type lockPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type legacyLockPackage struct {
	Version      string                     `json:"version"`
	Dependencies map[string]json.RawMessage `json:"dependencies"`
}

type installedPackage struct {
	Name     string
	Version  string
	Location string
}

var vulnerabilityIDPattern = regexp.MustCompile(
	`(?i)(CVE-\d{4}-\d+|GHSA-[A-Z0-9-]+)`,
)

func extractVulnerabilityID(values ...string) string {
	for _, value := range values {
		match := vulnerabilityIDPattern.FindString(value)

		if match != "" {
			return strings.ToUpper(match)
		}
	}

	return ""
}

func findingID(item advisory, packageName string) string {
	if item.Source > 0 {
		return fmt.Sprintf("npm:%d", item.Source)
	}

	if id := extractVulnerabilityID(
		item.URL,
		item.Title,
	); id != "" {
		return id
	}

	if item.URL != "" {
		return "npm:" + item.URL
	}

	return fmt.Sprintf(
		"npm:%s:%s",
		packageName,
		item.Title,
	)
}

func packageNameFromLocation(location string) string {
	location = strings.ReplaceAll(location, "\\", "/")

	parts := strings.Split(
		strings.Trim(location, "/"),
		"/",
	)

	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "node_modules" ||
			i+1 >= len(parts) {
			continue
		}

		name := parts[i+1]

		if strings.HasPrefix(name, "@") &&
			i+2 < len(parts) {
			name += "/" + parts[i+2]
		}

		return name
	}

	return ""
}
func loadLockPackages(
	lockPath string,
) (map[string]installedPackage, error) {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("read npm lockfile: %w", err)
	}

	return loadLockPackagesFromContent(content)
}
func loadLockPackagesFromContent(
	content []byte,
) (map[string]installedPackage, error) {
	result := make(map[string]installedPackage)

	var lock lockFile

	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, fmt.Errorf("parse npm lockfile: %w", err)
	}

	// package-lock v2/v3
	for location, item := range lock.Packages {
		if location == "" || item.Version == "" {
			continue
		}

		location = strings.TrimPrefix(
			strings.ReplaceAll(location, "\\", "/"),
			"./",
		)

		name := item.Name

		if name == "" {
			name = packageNameFromLocation(location)
		}

		result[location] = installedPackage{
			Name:     name,
			Version:  item.Version,
			Location: location,
		}
	}

	// package-lock v1 compatibility.
	var walk func(
		map[string]json.RawMessage,
		string,
	)

	walk = func(
		dependencies map[string]json.RawMessage,
		parent string,
	) {
		for name, raw := range dependencies {
			var item legacyLockPackage

			if err := json.Unmarshal(raw, &item); err != nil {
				continue
			}

			location := "node_modules/" + name

			if parent != "" {
				location = parent + "/node_modules/" + name
			}

			if item.Version != "" {
				result[location] = installedPackage{
					Name:     name,
					Version:  item.Version,
					Location: location,
				}
			}

			walk(item.Dependencies, location)
		}
	}

	walk(lock.Dependencies, "")

	return result, nil
}

func resolveInstalledPackages(
	lockPackages map[string]installedPackage,
	packageName string,
	nodes []string,
) []installedPackage {
	resolved := make([]installedPackage, 0)
	seen := make(map[string]bool)

	for _, node := range nodes {
		node = strings.TrimPrefix(
			strings.ReplaceAll(node, "\\", "/"),
			"./",
		)

		item, exists := lockPackages[node]

		if !exists {
			item = installedPackage{
				Name:     packageName,
				Location: node,
			}
		}

		key := item.Location

		if key == "" {
			key = packageName + "|" + item.Version
		}

		if seen[key] {
			continue
		}

		seen[key] = true
		resolved = append(resolved, item)
	}

	// Fallback for reports without node locations.
	if len(resolved) == 0 {
		for _, item := range lockPackages {
			if item.Name != packageName {
				continue
			}

			if !seen[item.Location] {
				seen[item.Location] = true
				resolved = append(resolved, item)
			}
		}
	}

	// Last fallback: preserve the vulnerability even when
	// an installed version cannot be resolved.
	if len(resolved) == 0 {
		resolved = append(resolved, installedPackage{
			Name: packageName,
		})
	}

	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].Location < resolved[j].Location
	})

	return resolved
}

func parseFindings(
	report auditReport,
	lockPath string,
	manifest string,
) ([]depscan.Finding, error) {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("read npm lockfile: %w", err)
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
) ([]depscan.Finding, error) {
	lockPackages, err := loadLockPackagesFromContent(lockContent)
	if err != nil {
		return nil, err
	}

	return parseFindingsWithPackages(
		report,
		lockPackages,
		manifest,
	), nil
}

func parseFindingsWithPackages(
	report auditReport,
	lockPackages map[string]installedPackage,
	manifest string,
) []depscan.Finding {

	findings := make([]depscan.Finding, 0)

	packageNames := make([]string, 0, len(report.Vulnerabilities))

	for name := range report.Vulnerabilities {
		packageNames = append(packageNames, name)
	}

	sort.Strings(packageNames)

	for _, packageName := range packageNames {
		vulnerability := report.Vulnerabilities[packageName]

		advisories := make([]advisory, 0)

		for _, raw := range vulnerability.Via {
			var item advisory

			// "via" may contain strings representing
			// dependency relationships. Only objects
			// represent actual security advisories.
			if len(raw) == 0 || raw[0] != '{' {
				continue
			}

			if err := json.Unmarshal(raw, &item); err != nil {
				continue
			}

			if item.Source == 0 &&
				item.Title == "" &&
				item.URL == "" {
				continue
			}

			advisories = append(advisories, item)
		}

		installedPackages := resolveInstalledPackages(
			lockPackages,
			packageName,
			vulnerability.Nodes,
		)

		for _, item := range advisories {
			id := findingID(item, packageName)

			title := item.Title

			if title == "" {
				title = "Security advisory affecting " + packageName
			}

			affectedRange := item.Range

			if affectedRange == "" {
				affectedRange = vulnerability.Range
			}

			vulnerabilityID := extractVulnerabilityID(
				item.URL,
				item.Title,
			)

			for _, installed := range installedPackages {
				metadata := map[string]any{
					"ecosystem":       "npm",
					"manifest_file":   manifest,
					"location":        installed.Location,
					"affected_range":  affectedRange,
					"is_direct":       vulnerability.IsDirect,
					"advisory_source": item.Source,
					"cwe":             item.CWE,
					"cvss": map[string]any{
						"score":         item.CVSS.Score,
						"vector_string": item.CVSS.VectorString,
					},
					"fix_available": vulnerability.FixAvailable,
					"effects":       vulnerability.Effects,
				}

				findings = append(findings, depscan.Finding{
					FindingType:      "vulnerability",
					FindingID:        id,
					VulnerabilityID:  vulnerabilityID,
					PackageName:      packageName,
					InstalledVersion: installed.Version,
					Severity: depscan.NormalizeSeverity(
						vulnerability.Severity,
					),
					Title:       title,
					Description: affectedRange,
					PrimaryURL:  item.URL,
					Metadata:    metadata,
				})
			}
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		left := findings[i]
		right := findings[j]

		if left.PackageName != right.PackageName {
			return left.PackageName < right.PackageName
		}

		if left.FindingID != right.FindingID {
			return left.FindingID < right.FindingID
		}

		return fmt.Sprint(
			left.Metadata["location"],
		) < fmt.Sprint(
			right.Metadata["location"],
		)
	})

	return findings
}

func lockfileName(directory string) string {
	if fileExists(filepath.Join(directory, "npm-shrinkwrap.json")) {
		return "npm-shrinkwrap.json"
	}

	if fileExists(filepath.Join(directory, "package-lock.json")) {
		return "package-lock.json"
	}

	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
