package depscan

import (
	"path"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
)

type MountCandidate struct {
	HostPath string
	Runtime  Runtime
}

type sourceGroup struct {
	source       Source
	runtimeIndex map[string]bool
}

func sourcePathKey(hostPath string) string {
	clean := filepath.Clean(hostPath)

	// Windows paths are case-insensitive.
	if goruntime.GOOS == "windows" {
		return strings.ToLower(clean)
	}

	return clean
}

func runtimeKey(runtime Runtime) string {
	return strings.ToLower(
		runtime.ContainerID + "|" +
			runtime.ContainerPath,
	)
}
func isDependencyDirectory(hostPath string) bool {
	directory := strings.ToLower(
		filepath.Base(filepath.Clean(hostPath)),
	)

	switch directory {
	case "node_modules", "vendor", ".git", "__pycache__", ".venv", "venv":
		return true
	default:
		return false
	}
}

// GroupSources groups runtime containers by their shared
// host source directory.
func GroupSources(
	projectDirectory string,
	candidates []MountCandidate,
) []Source {
	groups := make(map[string]*sourceGroup)

	for _, candidate := range candidates {
		if candidate.HostPath == "" ||
			candidate.Runtime.ContainerID == "" ||
			candidate.Runtime.ContainerPath == "" {
			continue
		}

		hostPath := filepath.Clean(candidate.HostPath)

		if isDependencyDirectory(hostPath) {
			continue
		}

		key := sourcePathKey(hostPath)

		group, exists := groups[key]

		if !exists {
			relativeDirectory := ""

			if projectDirectory != "" {
				relative, err := filepath.Rel(
					filepath.Clean(projectDirectory),
					hostPath,
				)

				if err == nil {
					relativeDirectory = filepath.ToSlash(relative)

					if relativeDirectory == "." {
						relativeDirectory = ""
					}
				}
			}

			group = &sourceGroup{
				source: Source{
					Directory:         hostPath,
					RelativeDirectory: relativeDirectory,
					Runtimes:          make([]Runtime, 0),
				},
				runtimeIndex: make(map[string]bool),
			}

			groups[key] = group
		}

		runtimeID := runtimeKey(candidate.Runtime)

		if group.runtimeIndex[runtimeID] {
			continue
		}

		group.runtimeIndex[runtimeID] = true

		group.source.Runtimes = append(
			group.source.Runtimes,
			candidate.Runtime,
		)
	}

	result := make([]Source, 0, len(groups))

	for _, group := range groups {
		sort.Slice(
			group.source.Runtimes,
			func(i, j int) bool {
				left := group.source.Runtimes[i]
				right := group.source.Runtimes[j]

				if left.ServiceName != right.ServiceName {
					return left.ServiceName < right.ServiceName
				}

				return left.ContainerID < right.ContainerID
			},
		)

		result = append(result, group.source)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Directory < result[j].Directory
	})

	return result
}

// ManifestPath returns a normalized path relative to the
// Docker Compose project directory.
func (s Source) ManifestPath(filename string) string {
	if s.RelativeDirectory == "" {
		return filepath.ToSlash(filename)
	}

	return path.Join(
		s.RelativeDirectory,
		filepath.ToSlash(filename),
	)
}
