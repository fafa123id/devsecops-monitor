package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Original Trivy JSON report.
type TrivyReport struct {
	ArtifactName string        `json:"ArtifactName"`
	ArtifactType string        `json:"ArtifactType"`
	Results      []TrivyResult `json:"Results"`
}

type TrivyResult struct {
	Target          string               `json:"Target"`
	Class           string               `json:"Class"`
	Type            string               `json:"Type"`
	Vulnerabilities []TrivyVulnerability `json:"Vulnerabilities"`
}

type TrivyVulnerability struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
	Description      string `json:"Description"`
	PrimaryURL       string `json:"PrimaryURL"`
	Status           string `json:"Status"`
}

// Flattened finding for application processing.
type TrivyFinding struct {
	TargetIdentifier string `json:"target_identifier"`
	TargetClass      string `json:"target_class"`
	TargetType       string `json:"target_type"`

	VulnerabilityID  string `json:"vulnerability_id"`
	PackageName      string `json:"package_name"`
	InstalledVersion string `json:"installed_version"`
	FixedVersion     string `json:"fixed_version"`

	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PrimaryURL  string `json:"primary_url"`
	Status      string `json:"status"`
}

type TrivySummary struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Unknown  int
	Total    int
}

// Parse JSON from any reader: file, stdout, or buffer.
func parseTrivyReport(reader io.Reader) (*TrivyReport, error) {
	var report TrivyReport

	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&report); err != nil {
		return nil, fmt.Errorf("decode Trivy JSON: %w", err)
	}

	return &report, nil
}

// Convert Trivy results into a flat list of findings.
func extractTrivyFindings(report *TrivyReport) []TrivyFinding {
	var findings []TrivyFinding

	if report == nil {
		return findings
	}

	for _, result := range report.Results {
		for _, vuln := range result.Vulnerabilities {
			findings = append(findings, TrivyFinding{
				TargetIdentifier: result.Target,
				TargetClass:      result.Class,
				TargetType:       result.Type,

				VulnerabilityID:  vuln.VulnerabilityID,
				PackageName:      vuln.PkgName,
				InstalledVersion: vuln.InstalledVersion,
				FixedVersion:     vuln.FixedVersion,

				Severity:    strings.ToUpper(vuln.Severity),
				Title:       vuln.Title,
				Description: vuln.Description,
				PrimaryURL:  vuln.PrimaryURL,
				Status:      vuln.Status,
			})
		}
	}

	return findings
}

func summarizeTrivyReport(report *TrivyReport) TrivySummary {
	var summary TrivySummary

	for _, result := range report.Results {
		for _, vuln := range result.Vulnerabilities {
			summary.Total++

			switch strings.ToUpper(vuln.Severity) {
			case "CRITICAL":
				summary.Critical++
			case "HIGH":
				summary.High++
			case "MEDIUM":
				summary.Medium++
			case "LOW":
				summary.Low++
			default:
				summary.Unknown++
			}
		}
	}

	return summary
}

func printTrivySummary(report *TrivyReport) {
	summary := summarizeTrivyReport(report)

	fmt.Println("\nTrivy Scan Summary")
	fmt.Println("--------------------------------")
	fmt.Printf("Artifact:  %s\n", report.ArtifactName)
	fmt.Printf("Type:      %s\n", report.ArtifactType)
	fmt.Printf("Targets:   %d\n", len(report.Results))
	fmt.Println("--------------------------------")
	fmt.Printf("Critical:  %d\n", summary.Critical)
	fmt.Printf("High:      %d\n", summary.High)
	fmt.Printf("Medium:    %d\n", summary.Medium)
	fmt.Printf("Low:       %d\n", summary.Low)
	fmt.Printf("Unknown:   %d\n", summary.Unknown)
	fmt.Println("--------------------------------")
	fmt.Printf("Total:     %d\n", summary.Total)
}

func scanDockerImage(ctx context.Context, imageReference string) (*TrivyReport, error) {
	if strings.TrimSpace(imageReference) == "" {
		return nil, fmt.Errorf("image reference cannot be empty")
	}

	fmt.Printf("Starting Trivy scan: %s\n", imageReference)

	cmd := exec.CommandContext(
		ctx,
		"trivy",
		"image",
		"--scanners", "vuln",
		"--format", "json",
		imageReference,
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf(
				"Trivy scan failed: %w; stderr: %s",
				err,
				strings.TrimSpace(string(exitErr.Stderr)),
			)
		}

		return nil, fmt.Errorf("execute Trivy: %w", err)
	}

	report, err := parseTrivyReport(bytes.NewReader(output))
	if err != nil {
		return nil, fmt.Errorf("parse Trivy output: %w", err)
	}

	return report, nil
}
