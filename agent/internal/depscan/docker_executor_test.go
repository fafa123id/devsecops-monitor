package depscan

import (
	"context"
	"strings"
	"testing"
)

func TestDockerExecutorRejectsEmptyRuntime(t *testing.T) {
	executor := NewDockerExecutor()

	_, err := executor.Execute(
		context.Background(),
		Runtime{},
		"composer",
		"audit",
	)

	if err == nil {
		t.Fatal("expected error for empty runtime")
	}

	if !strings.Contains(err.Error(), "container ID is empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDockerExecutorRejectsEmptyExecutable(t *testing.T) {
	executor := NewDockerExecutor()

	runtime := Runtime{
		ContainerID: "test-container",
	}

	_, err := executor.Execute(
		context.Background(),
		runtime,
		"",
		"audit",
	)

	if err == nil {
		t.Fatal("expected error for empty executable")
	}
}

func TestDockerExecutorProbeRejectsEmptyExecutable(t *testing.T) {
	executor := NewDockerExecutor()

	runtime := Runtime{
		ContainerID: "test-container",
	}

	err := executor.Probe(
		context.Background(),
		runtime,
		"",
	)

	if err == nil {
		t.Fatal("expected error for empty executable")
	}

}
func TestDockerExecutorFileExistsRejectsEmptyRuntime(t *testing.T) {
	executor := NewDockerExecutor()

	exists, err := executor.FileExists(
		context.Background(),
		Runtime{},
		"composer.json",
	)

	if err == nil {
		t.Fatal("expected error for empty runtime")
	}

	if exists {
		t.Fatal("expected file not to exist")
	}
}

func TestDockerExecutorReadFileRejectsEmptyRuntime(t *testing.T) {
	executor := NewDockerExecutor()

	_, err := executor.ReadFile(
		context.Background(),
		Runtime{},
		"composer.lock",
	)

	if err == nil {
		t.Fatal("expected error for empty runtime")
	}
}

func TestResolveContainerFilePath(t *testing.T) {
	runtime := Runtime{
		ContainerID:   "test-container",
		ContainerPath: "/var/www/app",
	}

	result, err := resolveContainerFilePath(
		runtime,
		"composer.lock",
	)

	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}

	if result != "/var/www/app/composer.lock" {
		t.Fatalf("unexpected path: %s", result)
	}
}

func TestResolveContainerFilePathRejectsTraversal(t *testing.T) {
	runtime := Runtime{
		ContainerID:   "test-container",
		ContainerPath: "/var/www/app",
	}

	_, err := resolveContainerFilePath(
		runtime,
		"../../etc/passwd",
	)

	if err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}
