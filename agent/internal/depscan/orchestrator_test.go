package depscan

import (
	"context"
	"errors"
	"testing"
)

type testExecutor struct {
	available  map[string]bool
	probeCalls []string
}

func (e *testExecutor) FileExists(
	context.Context,
	Runtime,
	string,
) (bool, error) {
	return false, nil
}

func (e *testExecutor) ReadFile(
	context.Context,
	Runtime,
	string,
) ([]byte, error) {
	return nil, nil
}
func (e *testExecutor) Probe(
	ctx context.Context,
	runtime Runtime,
	executable string,
) error {
	key := runtime.ContainerID + ":" + executable

	e.probeCalls = append(e.probeCalls, key)

	if e.available[key] {
		return nil
	}

	return errors.New("executable not available")
}

func (e *testExecutor) Execute(
	context.Context,
	Runtime,
	string,
	...string,
) (CommandResult, error) {
	return CommandResult{}, nil
}

type testScanner struct {
	name            string
	required        string
	detection       Detection
	scanResult      EcosystemResult
	scanError       error
	selectedRuntime string
}

func (s *testScanner) Ecosystem() string {
	return s.name
}

func (s *testScanner) Detect(Source) Detection {
	return s.detection
}

func (s *testScanner) RequiredExecutable() string {
	return s.required
}

func (s *testScanner) Scan(
	_ context.Context,
	_ Source,
	runtime Runtime,
	_ Executor,
	_ Detection,
) (EcosystemResult, error) {
	s.selectedRuntime = runtime.ServiceName
	return s.scanResult, s.scanError
}

func newTestOrchestrator(
	t *testing.T,
	scanners []Scanner,
	executor Executor,
) *Orchestrator {
	t.Helper()

	registry := NewRegistry()

	for _, scanner := range scanners {
		if err := registry.Register(scanner); err != nil {
			t.Fatalf("register test scanner: %v", err)
		}
	}

	orchestrator, err := NewOrchestrator(
		registry,
		executor,
	)
	if err != nil {
		t.Fatalf("create orchestrator: %v", err)
	}

	return orchestrator
}

func TestOrchestratorSelectsRuntimeByExecutable(t *testing.T) {
	scanner := &testScanner{
		name:     "npm",
		required: "npm",
		detection: Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "package.json",
			LockFile:     "package-lock.json",
		},
		scanResult: EcosystemResult{
			Ecosystem: "npm",
			Status:    StatusCompleted,
			Findings:  []Finding{},
		},
	}

	executor := &testExecutor{
		available: map[string]bool{
			"php-container:npm":  false,
			"vite-container:npm": true,
		},
	}

	runtimes := []Runtime{
		{
			ContainerID: "php-container",
			ServiceName: "app",
		},
		{
			ContainerID: "vite-container",
			ServiceName: "vite",
		},
	}

	orchestrator := newTestOrchestrator(
		t,
		[]Scanner{scanner},
		executor,
	)

	results, err := orchestrator.Scan(
		context.Background(),
		Source{
			Directory: "D:/projects/demo",
			Runtimes:  runtimes,
		},
	)

	if err != nil {
		t.Fatalf("orchestrator scan: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf(
			"expected 1 result, got %d",
			len(results),
		)
	}

	if results[0].Status != StatusCompleted {
		t.Errorf(
			"expected completed status, got %s",
			results[0].Status,
		)
	}

	if scanner.selectedRuntime != "vite" {
		t.Errorf(
			"expected vite runtime, got %s",
			scanner.selectedRuntime,
		)
	}

	if len(executor.probeCalls) != 2 {
		t.Errorf(
			"expected 2 runtime probes, got %d",
			len(executor.probeCalls),
		)
	}
}

func TestOrchestratorHandlesUnsupportedDetection(t *testing.T) {
	scanner := &testScanner{
		name:     "composer",
		required: "composer",
		detection: Detection{
			Present:           true,
			Supported:         false,
			ManifestFile:      "composer.json",
			UnsupportedReason: "composer.lock not found",
		},
	}

	executor := &testExecutor{
		available: map[string]bool{},
	}

	orchestrator := newTestOrchestrator(
		t,
		[]Scanner{scanner},
		executor,
	)

	results, err := orchestrator.Scan(
		context.Background(),
		Source{
			Directory: "D:/projects/demo",
		},
	)

	if err != nil {
		t.Fatalf("orchestrator scan: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != StatusUnsupported {
		t.Errorf(
			"expected unsupported, got %s",
			results[0].Status,
		)
	}

	if len(executor.probeCalls) != 0 {
		t.Error("unsupported detection should not probe runtime")
	}
}

func TestOrchestratorHandlesMissingRuntime(t *testing.T) {
	scanner := &testScanner{
		name:     "composer",
		required: "composer",
		detection: Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "composer.lock",
			LockFile:     "composer.lock",
		},
	}

	executor := &testExecutor{
		available: map[string]bool{},
	}

	orchestrator := newTestOrchestrator(
		t,
		[]Scanner{scanner},
		executor,
	)

	results, err := orchestrator.Scan(
		context.Background(),
		Source{
			Directory: "D:/projects/demo",
			Runtimes: []Runtime{
				{
					ContainerID: "node-container",
					ServiceName: "vite",
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("orchestrator scan: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != StatusFailed {
		t.Errorf(
			"expected failed, got %s",
			results[0].Status,
		)
	}
}

func TestOrchestratorSkipsUndetectedEcosystem(t *testing.T) {
	scanner := &testScanner{
		name:     "npm",
		required: "npm",
		detection: Detection{
			Present: false,
		},
	}

	executor := &testExecutor{
		available: map[string]bool{},
	}

	orchestrator := newTestOrchestrator(
		t,
		[]Scanner{scanner},
		executor,
	)

	results, err := orchestrator.Scan(
		context.Background(),
		Source{
			Directory: "D:/projects/demo",
		},
	)

	if err != nil {
		t.Fatalf("orchestrator scan: %v", err)
	}

	if len(results) != 0 {
		t.Errorf(
			"expected no results, got %d",
			len(results),
		)
	}
}

func TestOrchestratorHandlesScannerError(t *testing.T) {
	scanner := &testScanner{
		name:     "composer",
		required: "composer",
		detection: Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: "composer.lock",
		},
		scanError: errors.New("audit parsing failed"),
	}

	executor := &testExecutor{
		available: map[string]bool{
			"php-container:composer": true,
		},
	}

	orchestrator := newTestOrchestrator(
		t,
		[]Scanner{scanner},
		executor,
	)

	results, err := orchestrator.Scan(
		context.Background(),
		Source{
			Directory: "D:/projects/demo",
			Runtimes: []Runtime{
				{
					ContainerID: "php-container",
					ServiceName: "app",
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("orchestrator scan: %v", err)
	}

	if results[0].Status != StatusFailed {
		t.Errorf(
			"expected failed, got %s",
			results[0].Status,
		)
	}

	if results[0].ErrorMessage != "audit parsing failed" {
		t.Errorf(
			"unexpected error message: %s",
			results[0].ErrorMessage,
		)
	}
}

func TestNewOrchestratorRejectsNilDependencies(t *testing.T) {
	if _, err := NewOrchestrator(nil, nil); err == nil {
		t.Error("expected error for nil dependencies")
	}
}
