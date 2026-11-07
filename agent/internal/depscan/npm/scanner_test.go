package npm

import (
	"context"
	"encoding/json"
	"errors"
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
		return nil, errors.New("mock file not found")
	}

	return content, nil
}
func (m *mockExecutor) Execute(
	context.Context,
	depscan.Runtime,
	string,
	...string,
) (depscan.CommandResult, error) {
	return m.result, m.err
}

func TestParseNPMV3LockWithDependencyVersionStrings(t *testing.T) {
	directory := t.TempDir()
	lockPath := filepath.Join(directory, "package-lock.json")

	content := `{
		"lockfileVersion": 3,
		"packages": {
			"": {
				"dependencies": {
					"chart.js": "^4.4.0"
				}
			},
			"node_modules/chart.js": {
				"version": "4.4.0"
			}
		}
	}`

	if err := os.WriteFile(
		lockPath,
		[]byte(content),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	packages, err := loadLockPackages(lockPath)
	if err != nil {
		t.Fatalf("load lock packages: %v", err)
	}

	pkg, exists := packages["node_modules/chart.js"]
	if !exists {
		t.Fatal("chart.js package not found")
	}

	if pkg.Version != "4.4.0" {
		t.Errorf("expected 4.4.0, got %s", pkg.Version)
	}
}
func writeTestFile(
	t *testing.T,
	directory string,
	name string,
	content string,
) {
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

func TestDetectNPM(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "package.json", `{}`)
	writeTestFile(t, directory, "package-lock.json", `{
		"name": "demo",
		"lockfileVersion": 3,
		"packages": {}
	}`)

	source := depscan.Source{
		Directory:         directory,
		RelativeDirectory: "../frontend",
	}

	detection := New().Detect(source)

	if !detection.Present || !detection.Supported {
		t.Fatal("expected supported npm project")
	}

	if detection.LockFile != "package-lock.json" {
		t.Errorf(
			"unexpected lockfile: %s",
			detection.LockFile,
		)
	}

	if detection.ManifestFile != "../frontend/package-lock.json" {
		t.Errorf(
			"unexpected manifest path: %s",
			detection.ManifestFile,
		)
	}
}

func TestDetectNPMMissingLockfile(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "package.json", `{}`)

	detection := New().Detect(depscan.Source{
		Directory: directory,
	})

	if !detection.Present {
		t.Fatal("expected package.json to be detected")
	}

	if detection.Supported {
		t.Fatal("expected unsupported without npm lockfile")
	}

	if detection.ManifestFile != "package.json" {
		t.Errorf(
			"unexpected manifest file: %s",
			detection.ManifestFile,
		)
	}
}

func TestDetectNPMUsesShrinkwrap(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "package.json", `{}`)
	writeTestFile(t, directory, "package-lock.json", `{}`)
	writeTestFile(t, directory, "npm-shrinkwrap.json", `{
		"lockfileVersion": 3,
		"packages": {}
	}`)

	detection := New().Detect(depscan.Source{
		Directory: directory,
	})

	if detection.LockFile != "npm-shrinkwrap.json" {
		t.Errorf(
			"expected shrinkwrap, got %s",
			detection.LockFile,
		)
	}
}

func TestScanNPMWithVulnerability(t *testing.T) {
	directory := t.TempDir()

	writeTestFile(t, directory, "package.json", `{}`)

	writeTestFile(t, directory, "package-lock.json", `{
		"name": "demo",
		"lockfileVersion": 3,
		"packages": {
			"": {
				"name": "demo",
				"version": "1.0.0"
			},
			"node_modules/lodash": {
				"version": "4.17.20"
			}
		}
	}`)
	lockContent, err := os.ReadFile(
		filepath.Join(directory, "package-lock.json"),
	)
	if err != nil {
		t.Fatalf("read test lockfile: %v", err)
	}

	auditJSON := `{
		"auditReportVersion": 2,
		"vulnerabilities": {
			"lodash": {
				"name": "lodash",
				"severity": "high",
				"isDirect": true,
				"via": [
					{
						"source": 1106913,
						"name": "lodash",
						"dependency": "lodash",
						"title": "Command Injection",
						"url": "https://github.com/advisories/GHSA-test-1234",
						"range": "<4.17.21",
						"cwe": ["CWE-77"],
						"cvss": {
							"score": 7.2,
							"vectorString": "CVSS:3.1/example"
						}
					}
				],
				"effects": [],
				"range": "<4.17.21",
				"nodes": [
					"node_modules/lodash"
				],
				"fixAvailable": true
			}
		}
	}`

	executor := &mockExecutor{
		result: depscan.CommandResult{
			Stdout:   []byte(auditJSON),
			ExitCode: 1,
		},
		err: errors.New("exit status 1"),
		files: map[string][]byte{
			"package.json":      []byte(`{}`),
			"package-lock.json": lockContent,
		},
	}
	source := depscan.Source{
		Directory: directory,
		Runtimes: []depscan.Runtime{
			{
				ContainerID:   "vite-container",
				ContainerName: "digisejahtera_vite",
				ServiceName:   "vite",
				ContainerPath: "/var/www/app",
			},
		},
	}

	scanner := New()
	detection, err := scanner.DetectWithExecutor(
		context.Background(),
		source,
		executor,
	)
	if err != nil {
		t.Fatalf("detect npm through container: %v", err)
	}

	result, err := scanner.Scan(
		context.Background(),
		source,
		depscan.Runtime{
			ContainerID:   "vite-container",
			ServiceName:   "vite",
			ContainerPath: "/var/www/app",
		},
		executor,
		detection,
	)

	if err != nil {
		t.Fatalf("scan npm: %v", err)
	}

	if result.Status != depscan.StatusCompleted {
		t.Fatalf(
			"expected completed, got %s: %s",
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

	if finding.FindingID != "npm:1106913" {
		t.Errorf(
			"unexpected finding ID: %s",
			finding.FindingID,
		)
	}

	if finding.VulnerabilityID != "GHSA-TEST-1234" {
		t.Errorf(
			"unexpected vulnerability ID: %s",
			finding.VulnerabilityID,
		)
	}

	if finding.PackageName != "lodash" {
		t.Errorf(
			"unexpected package name: %s",
			finding.PackageName,
		)
	}

	if finding.InstalledVersion != "4.17.20" {
		t.Errorf(
			"unexpected installed version: %s",
			finding.InstalledVersion,
		)
	}

	if finding.Severity != "high" {
		t.Errorf(
			"unexpected severity: %s",
			finding.Severity,
		)
	}

	if finding.Metadata["location"] != "node_modules/lodash" {
		t.Errorf(
			"unexpected package location: %v",
			finding.Metadata["location"],
		)
	}
}

func TestParseEmptyNPMVulnerabilities(t *testing.T) {
	directory := t.TempDir()

	lockPath := filepath.Join(directory, "package-lock.json")

	if err := os.WriteFile(
		lockPath,
		[]byte(`{"lockfileVersion":3,"packages":{}}`),
		0600,
	); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	report := auditReport{
		AuditReportVersion: 2,
		Vulnerabilities:    map[string]vulnerability{},
	}

	findings, err := parseFindings(
		report,
		lockPath,
		"package-lock.json",
	)

	if err != nil {
		t.Fatalf("parse npm findings: %v", err)
	}

	if len(findings) != 0 {
		t.Errorf(
			"expected 0 findings, got %d",
			len(findings),
		)
	}
}

func TestParseNPMMultipleInstalledVersions(t *testing.T) {
	directory := t.TempDir()

	lockPath := filepath.Join(directory, "package-lock.json")

	lock := `{
		"lockfileVersion": 3,
		"packages": {
			"node_modules/lodash": {
				"version": "4.17.20"
			},
			"node_modules/parent/node_modules/lodash": {
				"version": "4.17.19"
			}
		}
	}`

	if err := os.WriteFile(
		lockPath,
		[]byte(lock),
		0600,
	); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	report := auditReport{
		Vulnerabilities: map[string]vulnerability{
			"lodash": {
				Name:     "lodash",
				Severity: "high",
				Via: []json.RawMessage{
					[]byte(`{
						"source": 1106913,
						"title": "Command Injection",
						"url": "https://example.com/advisory"
					}`),
				},
				Nodes: []string{
					"node_modules/lodash",
					"node_modules/parent/node_modules/lodash",
				},
			},
		},
	}

	findings, err := parseFindings(
		report,
		lockPath,
		"package-lock.json",
	)

	if err != nil {
		t.Fatalf("parse npm findings: %v", err)
	}

	if len(findings) != 2 {
		t.Fatalf(
			"expected 2 installed package findings, got %d",
			len(findings),
		)
	}

	versions := map[string]bool{}

	for _, finding := range findings {
		versions[finding.InstalledVersion] = true
	}

	if !versions["4.17.20"] || !versions["4.17.19"] {
		t.Errorf(
			"expected both installed versions, got %v",
			versions,
		)
	}
}

func TestParseNPMScopedPackageLocation(t *testing.T) {
	name := packageNameFromLocation(
		"node_modules/@example/library",
	)

	if name != "@example/library" {
		t.Errorf("unexpected scoped package name: %s", name)
	}
}
