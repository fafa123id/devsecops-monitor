package golang

import (
	"testing"
)

func TestParseGovulncheckJSON(t *testing.T) {
	// Intentionally place finding before OSV to verify that
	// parser does not depend on JSON message order.
	report := `
{"config":{"protocol_version":"v1.0.0","scan_level":"symbol"}}
{"finding":{"osv":"GO-2025-1234","fixed_version":"v0.3.1","trace":[{"module":"golang.org/x/text","version":"v0.3.0","package":"golang.org/x/text/language","function":"Parse","position":{"filename":"main.go","line":12,"column":3}}]}}
{"osv":{"id":"GO-2025-1234","summary":"Example Go vulnerability","details":"Example vulnerability description","aliases":["CVE-2025-12345","GHSA-test-1234"],"database_specific":{"severity":"HIGH"},"references":[{"type":"WEB","url":"https://pkg.go.dev/vuln/GO-2025-1234"}]}}
{"SBOM":{"modules":[{"path":"golang.org/x/text","version":"v0.3.0"}]}}
`

	findings, err := parseFindings([]byte(report))
	if err != nil {
		t.Fatalf("parse govulncheck output: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	finding := findings[0]

	if finding.FindingID != "GO-2025-1234" {
		t.Errorf("unexpected finding ID: %s", finding.FindingID)
	}

	if finding.VulnerabilityID != "CVE-2025-12345" {
		t.Errorf("unexpected vulnerability ID: %s", finding.VulnerabilityID)
	}

	if finding.PackageName != "golang.org/x/text" {
		t.Errorf("unexpected package: %s", finding.PackageName)
	}

	if finding.InstalledVersion != "v0.3.0" {
		t.Errorf("unexpected version: %s", finding.InstalledVersion)
	}

	if finding.FixedVersion != "v0.3.1" {
		t.Errorf("unexpected fixed version: %s", finding.FixedVersion)
	}

	if finding.Severity != "high" {
		t.Errorf("unexpected severity: %s", finding.Severity)
	}

	if finding.Metadata["go_package"] != "golang.org/x/text/language" {
		t.Errorf("unexpected Go package: %v", finding.Metadata["go_package"])
	}

	if finding.Metadata["scan_level"] != "symbol" {
		t.Errorf("unexpected scan level: %v", finding.Metadata["scan_level"])
	}
}

func TestParseGovulncheckDeduplicatesSameModuleAndPackage(t *testing.T) {
	report := `
{"osv":{"id":"GO-2025-1000","summary":"Example vulnerability"}}
{"finding":{"osv":"GO-2025-1000","trace":[{"module":"example.com/library","package":"example.com/library/parser","function":"ParseA"}]}}
{"finding":{"osv":"GO-2025-1000","trace":[{"module":"example.com/library","package":"example.com/library/parser","function":"ParseB"}]}}
`

	findings, err := parseFindings([]byte(report))
	if err != nil {
		t.Fatalf("parse govulncheck output: %v", err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 canonical finding, got %d", len(findings))
	}

	traces, ok := findings[0].Metadata["traces"].([]goTraceFrame)
	if !ok {
		t.Fatal("expected trace metadata")
	}

	if len(traces) != 2 {
		t.Errorf("expected 2 unique traces, got %d", len(traces))
	}
}

func TestParseGovulncheckEmptyFindings(t *testing.T) {
	report := `
{"config":{"protocol_version":"v1.0.0","scan_level":"symbol"}}
{"SBOM":{"modules":[]}}
`

	findings, err := parseFindings([]byte(report))
	if err != nil {
		t.Fatalf("parse empty govulncheck report: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestParseGovulncheckRejectsInvalidJSON(t *testing.T) {
	_, err := parseFindings([]byte(`{"finding":`))

	if err == nil {
		t.Fatal("expected JSON parsing error")
	}
}
