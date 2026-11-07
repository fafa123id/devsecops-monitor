package main

import (
	"devsecops-agent/internal/depscan"
	"fmt"
	"strings"
)

type ScanSubmission struct {
	TriggerType    string               `json:"trigger_type"`
	ScannerVersion string               `json:"scanner_version"`
	Projects       []ProjectScanPayload `json:"projects"`
}

type ProjectScanPayload struct {
	DockerProjectName string                 `json:"docker_project_name"`
	Ecosystems        []EcosystemScanPayload `json:"ecosystems"`
}

type EcosystemScanPayload = depscan.EcosystemResult

type ScanFindingPayload = depscan.Finding

func normalizeSeverity(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium", "moderate":
		return "medium"
	case "low":
		return "low"
	default:
		return "unknown"
	}
}

func validateScanSubmission(submission ScanSubmission) error {
	if strings.TrimSpace(submission.ScannerVersion) == "" {
		return fmt.Errorf("scanner version is required")
	}
	switch submission.TriggerType {
	case "manual", "discovery", "scheduled", "image_updated", "remediation":
	default:
		return fmt.Errorf(
			"invalid scan trigger type: %q",
			submission.TriggerType,
		)
	}
	if len(submission.Projects) == 0 {
		return fmt.Errorf("at least one project is required")
	}

	for _, project := range submission.Projects {
		if strings.TrimSpace(project.DockerProjectName) == "" {
			return fmt.Errorf("docker project name is required")
		}

		if len(project.Ecosystems) == 0 {
			return fmt.Errorf(
				"project %q has no ecosystem scan results",
				project.DockerProjectName,
			)
		}

		for _, ecosystem := range project.Ecosystems {
			if strings.TrimSpace(ecosystem.Ecosystem) == "" {
				return fmt.Errorf(
					"ecosystem name is required for project %q",
					project.DockerProjectName,
				)
			}

			if strings.TrimSpace(ecosystem.ManifestFile) == "" {
				return fmt.Errorf(
					"manifest file is required for project %q",
					project.DockerProjectName,
				)
			}

			switch ecosystem.Status {
			case "completed", "failed", "unsupported":
			default:
				return fmt.Errorf(
					"invalid ecosystem status %q for project %q",
					ecosystem.Status,
					project.DockerProjectName,
				)
			}

			for _, finding := range ecosystem.Findings {
				if strings.TrimSpace(finding.FindingID) == "" {
					return fmt.Errorf(
						"finding ID is required for ecosystem %q",
						ecosystem.Ecosystem,
					)
				}
			}
		}
	}

	return nil
}
