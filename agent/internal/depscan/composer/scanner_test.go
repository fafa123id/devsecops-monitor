package composer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"devsecops-agent/internal/depscan"
)

type mockExecutor struct {
	result depscan.CommandResult
	err    error
	files  map[string][]byte
}

func (m *mockExecutor) Probe(
	context.Context,
	depscan.Runtime,
	string,
) error {
	return nil
}

func (m *mockExecutor) Execute(
	context.Context,
	depscan.Runtime,
	string,
	...string,
) (depscan.CommandResult, error) {
	return m.result, m.err
}

func (m *mockExecutor) FileExists(
	_ context.Context,
	_ depscan.Runtime,
	relativePath string,
) (bool, error) {
	_, exists := m.files[relativePath]
	return exists, nil
}

func (m *mockExecutor) ReadFile(
	_ context.Context,
	_ depscan.Runtime,
	relativePath string,
) ([]byte, error) {
	content, exists := m.files[relativePath]
	if !exists {
		return nil, fmt.Errorf("mock file %q not found", relativePath)
	}

	return content, nil
}
func writeTestFile(t *testing.T, directory, name, content string) {
	t.Helper()

	err := os.WriteFile(
		filepath.Join(directory, name),
		[]byte(content),
		0600,
	)

	if err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func TestDetectComposer(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "composer.json", `{}`)
	writeTestFile(t, directory, "composer.lock", `{"packages":[]}`)

	scanner := New()

	detection := scanner.Detect(depscan.Source{
		Directory: directory,
	})

	if !detection.Present || !detection.Supported {
		t.Fatalf("expected supported Composer project")
	}

	if detection.LockFile != "composer.lock" {
		t.Errorf(
			"unexpected lockfile: %s",
			detection.LockFile,
		)
	}
}

func TestDetectComposerMissingLock(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "composer.json", `{}`)

	detection := New().Detect(depscan.Source{
		Directory: directory,
	})

	if !detection.Present {
		t.Fatal("expected Composer manifest to be detected")
	}

	if detection.Supported {
		t.Fatal("expected project without lockfile to be unsupported")
	}
}

func TestScanComposerWithVulnerability(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "composer.json", `{}`)

	writeTestFile(t, directory, "composer.lock", `{
		"packages": [
			{
				"name": "symfony/http-kernel",
				"version": "v6.4.14"
			}
		],
		"packages-dev": []
	}`)
	lockContent, readErr := os.ReadFile(
		filepath.Join(directory, "composer.lock"),
	)
	if readErr != nil {
		t.Fatalf("read test lockfile: %v", readErr)
	}
	auditJSON := `{
		"advisories": {
			"symfony/http-kernel": [
				{
					"advisoryId": "GHSA-test-1234",
					"packageName": "symfony/http-kernel",
					"cve": "CVE-2025-12345",
					"title": "Symfony HTTP Kernel vulnerability",
					"link": "https://example.com/advisory",
					"severity": "high",
					"affectedVersions": "<6.4.15"
				}
			]
		}
	}`

	executor := &mockExecutor{
		result: depscan.CommandResult{
			Stdout:   []byte(auditJSON),
			ExitCode: 1,
		},
		err: errors.New("exit status 1"),
		files: map[string][]byte{
			"composer.lock": lockContent,
		},
	}

	source := depscan.Source{
		Directory: directory,
	}

	scanner := New()
	detection := scanner.Detect(source)

	result, err := scanner.Scan(
		context.Background(),
		source,
		depscan.Runtime{},
		executor,
		detection,
	)

	if err != nil {
		t.Fatalf("scan Composer: %v", err)
	}

	if result.Status != depscan.StatusCompleted {
		t.Fatalf(
			"expected completed status, got %s: %s",
			result.Status,
			result.ErrorMessage,
		)
	}

	if len(result.Findings) != 1 {
		t.Fatalf(
			"expected 1 finding, got %d",
			len(result.Findings),
		)
	}

	finding := result.Findings[0]

	if finding.FindingID != "GHSA-test-1234" {
		t.Errorf("unexpected finding ID: %s", finding.FindingID)
	}

	if finding.InstalledVersion != "v6.4.14" {
		t.Errorf(
			"unexpected installed version: %s",
			finding.InstalledVersion,
		)
	}

	if finding.Severity != "high" {
		t.Errorf("unexpected severity: %s", finding.Severity)
	}
}

func TestParseEmptyComposerAdvisories(t *testing.T) {
	report := []byte(`{
		"advisories": [],
		"abandoned": [],
		"filter": [],
		"cooldown": []
	}`)

	var parsed auditReport

	if err := parsed.UnmarshalJSON(report); err != nil {
		t.Fatalf("parse Composer report: %v", err)
	}

	if len(parsed.Advisories) != 0 {
		t.Errorf(
			"expected empty advisories, got %d",
			len(parsed.Advisories),
		)
	}
}
func TestDetectComposerThroughContainer(t *testing.T) {
	executor := &mockExecutor{
		files: map[string][]byte{
			"composer.json": []byte(`{}`),
			"composer.lock": []byte(`{"packages":[]}`),
		},
	}

	source := depscan.Source{
		Directory: "D:/projects/demo",
		Runtimes: []depscan.Runtime{
			{
				ContainerID:   "php-container",
				ContainerName: "demo-app",
				ServiceName:   "app",
				ContainerPath: "/var/www/html",
			},
		},
	}

	detection, err := New().DetectWithExecutor(
		context.Background(),
		source,
		executor,
	)

	if err != nil {
		t.Fatalf("detect Composer through container: %v", err)
	}

	if !detection.Present || !detection.Supported {
		t.Fatalf("expected supported Composer project: %+v", detection)
	}

	if detection.LockFile != "composer.lock" {
		t.Errorf("unexpected lockfile: %s", detection.LockFile)
	}
}
