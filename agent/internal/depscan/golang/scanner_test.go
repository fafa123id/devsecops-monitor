package golang

import (
	"context"
	"errors"
	"testing"

	"devsecops-agent/internal/depscan"
)

type mockExecutor struct {
	files       map[string][]byte
	result      depscan.CommandResult
	executeErr  error
	probeErrors map[string]error
	probeCalls  []string
	executable  string
	args        []string
}

func (m *mockExecutor) Probe(
	_ context.Context,
	_ depscan.Runtime,
	executable string,
) error {
	m.probeCalls = append(m.probeCalls, executable)
	return m.probeErrors[executable]
}

func (m *mockExecutor) Execute(
	_ context.Context,
	_ depscan.Runtime,
	executable string,
	args ...string,
) (depscan.CommandResult, error) {
	m.executable = executable
	m.args = args

	return m.result, m.executeErr
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

func TestDetectGoModuleThroughContainer(t *testing.T) {
	executor := &mockExecutor{
		files: map[string][]byte{
			"go.mod": []byte("module example.com/demo\ngo 1.27\n"),
		},
	}

	source := depscan.Source{
		Directory: "D:/projects/demo",
		Runtimes: []depscan.Runtime{
			{
				ContainerID:   "go-container",
				ContainerName: "demo-app",
				ServiceName:   "app",
				ContainerPath: "/app",
			},
		},
	}

	detection, err := New().DetectWithExecutor(
		context.Background(),
		source,
		executor,
	)

	if err != nil {
		t.Fatalf("detect Go module: %v", err)
	}

	if !detection.Present || !detection.Supported {
		t.Fatalf("expected supported Go module: %+v", detection)
	}

	if detection.ManifestFile != "go.mod" {
		t.Errorf("unexpected manifest: %s", detection.ManifestFile)
	}
}

func TestDetectGoModuleNotPresent(t *testing.T) {
	executor := &mockExecutor{
		files: map[string][]byte{},
	}

	detection, err := New().DetectWithExecutor(
		context.Background(),
		depscan.Source{
			Runtimes: []depscan.Runtime{
				{
					ContainerID:   "app-container",
					ContainerPath: "/app",
				},
			},
		},
		executor,
	)

	if err != nil {
		t.Fatalf("detect Go module: %v", err)
	}

	if detection.Present {
		t.Fatal("expected Go module not to be detected")
	}
}

func TestScanGoWithVulnerability(t *testing.T) {
	report := `
{"config":{"protocol_version":"v1.0.0","scan_level":"symbol"}}
{"osv":{"id":"GO-2025-1234","summary":"Example Go vulnerability","aliases":["CVE-2025-12345"],"database_specific":{"severity":"HIGH"}}}
{"finding":{"osv":"GO-2025-1234","fixed_version":"v0.3.1","trace":[{"module":"golang.org/x/text","version":"v0.3.0","package":"golang.org/x/text/language","function":"Parse"}]}}
`

	executor := &mockExecutor{
		result: depscan.CommandResult{
			Stdout:   []byte(report),
			ExitCode: 0,
		},
		probeErrors: map[string]error{},
	}

	source := depscan.Source{
		Directory: "D:/projects/demo",
	}

	runtime := depscan.Runtime{
		ContainerID:   "go-container",
		ContainerName: "demo-app",
		ServiceName:   "app",
		ContainerPath: "/app",
	}

	result, err := New().Scan(
		context.Background(),
		source,
		runtime,
		executor,
		depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "go.mod",
			LockFile:     "go.mod",
		},
	)

	if err != nil {
		t.Fatalf("scan Go dependencies: %v", err)
	}

	if result.Status != depscan.StatusCompleted {
		t.Fatalf(
			"expected completed, got %s: %s",
			result.Status,
			result.ErrorMessage,
		)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}

	if executor.executable != "govulncheck" {
		t.Errorf("unexpected executable: %s", executor.executable)
	}

	if len(executor.args) != 2 ||
		executor.args[0] != "-json" ||
		executor.args[1] != "./..." {
		t.Errorf("unexpected govulncheck arguments: %v", executor.args)
	}

	if result.Findings[0].FindingID != "GO-2025-1234" {
		t.Errorf("unexpected finding: %+v", result.Findings[0])
	}
}

func TestScanGoRequiresGoToolchain(t *testing.T) {
	executor := &mockExecutor{
		probeErrors: map[string]error{
			"go": errors.New("go executable unavailable"),
		},
	}

	result, err := New().Scan(
		context.Background(),
		depscan.Source{},
		depscan.Runtime{
			ContainerID:   "app-container",
			ContainerPath: "/app",
		},
		executor,
		depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "go.mod",
		},
	)

	if err != nil {
		t.Fatalf("scan should return a failed result, not a fatal error: %v", err)
	}

	if result.Status != depscan.StatusFailed {
		t.Fatalf("expected failed status, got %s", result.Status)
	}
}
