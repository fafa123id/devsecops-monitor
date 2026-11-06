package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestParseTrivyReport(t *testing.T) {
	file, err := os.Open("../trivy-result.json")
	if err != nil {
		t.Fatalf("open Trivy JSON: %v", err)
	}
	defer file.Close()

	report, err := parseTrivyReport(file)
	if err != nil {
		t.Fatalf("parse Trivy JSON: %v", err)
	}

	if report.ArtifactName == "" {
		t.Fatal("artifact name is empty")
	}

	printTrivySummary(report)

	t.Logf("Parsed artifact: %s", report.ArtifactName)
}

func TestScanDockerImage(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Minute,
	)
	defer cancel()

	report, err := scanDockerImage(ctx, "deployment-api")
	if err != nil {
		t.Fatalf("scan Docker image: %v", err)
	}

	printTrivySummary(report)

	findings := extractTrivyFindings(report)

	t.Logf("Extracted findings: %d", len(findings))

	if len(findings) == 0 {
		t.Log("No vulnerabilities detected.")
	}
}
