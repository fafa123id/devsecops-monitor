package python

import (
	"context"
	"errors"
	"testing"

	"devsecops-agent/internal/depscan"
)

type mockExecutor struct {
	command depscan.CommandResult
	execErr error
}

func (m *mockExecutor) Probe(
	ctx context.Context,
	runtime depscan.Runtime,
	executable string,
) error {
	return nil
}

func (m *mockExecutor) Execute(
	ctx context.Context,
	runtime depscan.Runtime,
	executable string,
	args ...string,
) (depscan.CommandResult, error) {
	return m.command, m.execErr
}

func (m *mockExecutor) FileExists(
	ctx context.Context,
	runtime depscan.Runtime,
	relativePath string,
) (bool, error) {
	return true, nil
}

func (m *mockExecutor) ReadFile(
	ctx context.Context,
	runtime depscan.Runtime,
	relativePath string,
) ([]byte, error) {
	return nil, nil
}

func TestParseFindingsFromContent(t *testing.T) {
	content := []byte(`{
		"dependencies": [
			{
				"name": "django",
				"version": "4.2.0",
				"vulns": [
					{
						"id": "PYSEC-2024-1",
						"fix_versions": ["4.2.10"],
						"aliases": ["CVE-2024-12345"],
						"description": "Django security vulnerability"
					}
				]
			}
		]
	}`)

	findings, err := parseFindingsFromContent(
		content,
		"requirements.txt",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]

	if finding.PackageName != "django" {
		t.Errorf("expected package django, got %s", finding.PackageName)
	}

	if finding.InstalledVersion != "4.2.0" {
		t.Errorf("expected version 4.2.0, got %s", finding.InstalledVersion)
	}

	if finding.FindingID != "PYSEC-2024-1" {
		t.Errorf("unexpected finding ID: %s", finding.FindingID)
	}

	if finding.VulnerabilityID != "CVE-2024-12345" {
		t.Errorf("unexpected vulnerability ID: %s", finding.VulnerabilityID)
	}

	if finding.Severity != "unknown" {
		t.Errorf("expected unknown severity, got %s", finding.Severity)
	}

	if finding.Metadata["manifest_file"] != "requirements.txt" {
		t.Errorf("unexpected manifest: %v", finding.Metadata["manifest_file"])
	}
}

func TestParseFindingsFromContentWithoutVulnerabilities(t *testing.T) {
	content := []byte(`{
		"dependencies": [
			{
				"name": "requests",
				"version": "2.32.0",
				"vulns": []
			}
		]
	}`)

	findings, err := parseFindingsFromContent(
		content,
		"requirements.txt",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestParseFindingsFromContentInvalidJSON(t *testing.T) {
	_, err := parseFindingsFromContent(
		[]byte(`invalid json`),
		"requirements.txt",
	)

	if err == nil {
		t.Fatal("expected JSON parsing error")
	}
}

func TestScannerScanCompleted(t *testing.T) {
	executor := &mockExecutor{
		command: depscan.CommandResult{
			ExitCode: 0,
			Stdout: []byte(`{
				"dependencies": [
					{
						"name": "django",
						"version": "4.2.0",
						"vulns": [
							{
								"id": "PYSEC-2024-1",
								"fix_versions": ["4.2.10"],
								"aliases": ["CVE-2024-12345"],
								"description": "Django security vulnerability"
							}
						]
					}
				]
			}`),
		},
	}

	scanner := New()

	result, err := scanner.Scan(
		context.Background(),
		depscan.Source{},
		depscan.Runtime{},
		executor,
		depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "requirements.txt",
			LockFile:     "requirements.txt",
		},
	)

	if err != nil {
		t.Fatalf("unexpected scan error: %v", err)
	}

	if result.Status != depscan.StatusCompleted {
		t.Errorf("expected completed, got %s: %s", result.Status, result.ErrorMessage)
	}

	if len(result.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(result.Findings))
	}
}

func TestScannerScanExecutionFailure(t *testing.T) {
	executor := &mockExecutor{
		execErr: errors.New("pip-audit not found"),
	}

	scanner := New()

	result, err := scanner.Scan(
		context.Background(),
		depscan.Source{},
		depscan.Runtime{},
		executor,
		depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "requirements.txt",
			LockFile:     "requirements.txt",
		},
	)

	if err != nil {
		t.Fatalf("unexpected scan error: %v", err)
	}

	if result.Status != depscan.StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
}
