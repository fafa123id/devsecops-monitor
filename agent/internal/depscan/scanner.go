package depscan

import "context"

type Executor interface {
	Probe(
		ctx context.Context,
		runtime Runtime,
		executable string,
	) error

	Execute(
		ctx context.Context,
		runtime Runtime,
		executable string,
		args ...string,
	) (CommandResult, error)

	FileExists(
		ctx context.Context,
		runtime Runtime,
		relativePath string,
	) (bool, error)

	ReadFile(
		ctx context.Context,
		runtime Runtime,
		relativePath string,
	) ([]byte, error)
}

// Scanner defines the standard behavior
// required by every dependency scanner.
type Scanner interface {
	// Ecosystem returns the scanner identifier.
	// Examples: composer, npm, golang, rust, python.
	Ecosystem() string

	// Detect checks whether this ecosystem
	// exists and is supported in the source directory.
	Detect(source Source) Detection

	// RequiredExecutable returns the required
	// CLI executable for this scanner.
	RequiredExecutable() string

	// Scan executes the ecosystem audit
	// and returns normalized findings.
	Scan(
		ctx context.Context,
		source Source,
		runtime Runtime,
		executor Executor,
		detection Detection,
	) (EcosystemResult, error)
}

// ContainerDetector performs manifest detection through
// a runtime container instead of the agent filesystem.
type ContainerDetector interface {
	DetectWithExecutor(
		ctx context.Context,
		source Source,
		executor Executor,
	) (Detection, error)
}
