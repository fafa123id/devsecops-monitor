package python

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"devsecops-agent/internal/depscan"
)

type auditReport struct {
	Dependencies []dependency `json:"dependencies"`
}

type dependency struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Vulns   []vulnerability `json:"vulns"`
}

type vulnerability struct {
	ID          string   `json:"id"`
	FixVersions []string `json:"fix_versions"`
	Description string   `json:"description"`
	Aliases     []string `json:"aliases"`
}

func parseFindingsFromContent(
	content []byte,
	manifest string,
) ([]depscan.Finding, error) {
	var report auditReport

	if err := json.Unmarshal(content, &report); err != nil {
		return nil, fmt.Errorf("parse pip-audit report: %w", err)
	}

	findings := make([]depscan.Finding, 0)

	for _, dependency := range report.Dependencies {
		for _, vulnerability := range dependency.Vulns {
			id := strings.TrimSpace(vulnerability.ID)

			if id == "" {
				continue
			}

			vulnerabilityID := ""

			if strings.HasPrefix(strings.ToUpper(id), "CVE-") ||
				strings.HasPrefix(strings.ToUpper(id), "GHSA-") {
				vulnerabilityID = strings.ToUpper(id)
			}

			aliases := make([]string, 0, len(vulnerability.Aliases))

			for _, alias := range vulnerability.Aliases {
				alias = strings.TrimSpace(alias)

				if alias != "" {
					aliases = append(aliases, alias)

					if vulnerabilityID == "" &&
						(strings.HasPrefix(strings.ToUpper(alias), "CVE-") ||
							strings.HasPrefix(strings.ToUpper(alias), "GHSA-")) {
						vulnerabilityID = strings.ToUpper(alias)
					}
				}
			}

			sort.Strings(aliases)

			fixVersions := append(
				[]string(nil),
				vulnerability.FixVersions...,
			)
			sort.Strings(fixVersions)

			title := "Vulnerability affecting " + dependency.Name

			if vulnerability.Description != "" {
				title = vulnerability.Description
			}

			metadata := map[string]any{
				"ecosystem":      "pip",
				"manifest_file":  manifest,
				"aliases":        aliases,
				"fix_versions":   fixVersions,
				"affected_range": "",
			}

			findings = append(findings, depscan.Finding{
				FindingType:      "vulnerability",
				FindingID:        id,
				VulnerabilityID:  vulnerabilityID,
				PackageName:      dependency.Name,
				InstalledVersion: dependency.Version,
				Severity:         depscan.NormalizeSeverity(""),
				Title:            title,
				Description:      vulnerability.Description,
				Metadata:         metadata,
			})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].PackageName != findings[j].PackageName {
			return findings[i].PackageName < findings[j].PackageName
		}

		return findings[i].FindingID < findings[j].FindingID
	})

	return findings, nil
}
