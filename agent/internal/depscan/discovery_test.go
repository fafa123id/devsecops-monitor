package depscan

import (
	"path/filepath"
	"testing"
)

func TestGroupSources(t *testing.T) {
	root := t.TempDir()

	projectDir := filepath.Join(root, "deployment")
	backendDir := filepath.Join(root, "backend")
	frontendDir := filepath.Join(root, "frontend")

	candidates := []MountCandidate{
		{
			HostPath: backendDir,
			Runtime: Runtime{
				ContainerID:   "php-container",
				ServiceName:   "app",
				ContainerPath: "/var/www/html",
			},
		},
		{
			HostPath: backendDir,
			Runtime: Runtime{
				ContainerID:   "node-container",
				ServiceName:   "vite",
				ContainerPath: "/var/www/html",
			},
		},
		{
			HostPath: backendDir,
			Runtime: Runtime{
				ContainerID:   "php-container",
				ServiceName:   "app",
				ContainerPath: "/var/www/html",
			},
		},
		{
			HostPath: frontendDir,
			Runtime: Runtime{
				ContainerID:   "frontend-container",
				ServiceName:   "frontend",
				ContainerPath: "/app",
			},
		},
	}

	sources := GroupSources(projectDir, candidates)

	if len(sources) != 2 {
		t.Fatalf(
			"expected 2 source groups, got %d",
			len(sources),
		)
	}

	var backend *Source

	for i := range sources {
		if sources[i].Directory == backendDir {
			backend = &sources[i]
			break
		}
	}

	if backend == nil {
		t.Fatal("backend source directory not found")
	}

	if len(backend.Runtimes) != 2 {
		t.Fatalf(
			"expected 2 unique runtimes, got %d",
			len(backend.Runtimes),
		)
	}

	expectedRelative := filepath.ToSlash(
		filepath.Join("..", "backend"),
	)

	if backend.RelativeDirectory != expectedRelative {
		t.Errorf(
			"expected relative directory %s, got %s",
			expectedRelative,
			backend.RelativeDirectory,
		)
	}
}

func TestGroupSourcesIgnoresInvalidCandidates(t *testing.T) {
	sources := GroupSources(
		"D:/projects/deployment",
		[]MountCandidate{
			{
				HostPath: "",
				Runtime: Runtime{
					ContainerID:   "container-1",
					ContainerPath: "/app",
				},
			},
			{
				HostPath: "D:/projects/backend",
				Runtime: Runtime{
					ContainerID:   "container-2",
					ContainerPath: "",
				},
			},
		},
	)

	if len(sources) != 0 {
		t.Errorf(
			"expected no source groups, got %d",
			len(sources),
		)
	}
}

func TestSourceManifestPath(t *testing.T) {
	source := Source{
		Directory:         "D:/projects/backend",
		RelativeDirectory: "../backend",
	}

	result := source.ManifestPath("composer.lock")

	expected := "../backend/composer.lock"

	if result != expected {
		t.Errorf(
			"expected %s, got %s",
			expected,
			result,
		)
	}
}

func TestSourceManifestPathWithoutRelativeDirectory(t *testing.T) {
	source := Source{
		Directory: "D:/projects/backend",
	}

	result := source.ManifestPath("package-lock.json")

	if result != "package-lock.json" {
		t.Errorf(
			"unexpected manifest path: %s",
			result,
		)
	}
}
func TestGroupSourcesIgnoresDependencyDirectoryMounts(t *testing.T) {
	root := t.TempDir()

	candidates := []MountCandidate{
		{
			HostPath: filepath.Join(root, "backend"),
			Runtime: Runtime{
				ContainerID:   "app-container",
				ContainerPath: "/var/www/html",
			},
		},
		{
			HostPath: filepath.Join(root, "backend", "vendor"),
			Runtime: Runtime{
				ContainerID:   "vendor-container",
				ContainerPath: "/var/www/html/vendor",
			},
		},
		{
			HostPath: filepath.Join(root, "frontend", "node_modules"),
			Runtime: Runtime{
				ContainerID:   "node-container",
				ContainerPath: "/app/node_modules",
			},
		},
	}

	sources := GroupSources(root, candidates)

	if len(sources) != 1 {
		t.Fatalf(
			"expected 1 source group, got %d",
			len(sources),
		)
	}
}
