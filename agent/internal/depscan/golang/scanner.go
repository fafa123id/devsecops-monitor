package golang

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"devsecops-agent/internal/depscan"
)

type Scanner struct{}

func New() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Ecosystem() string {
	return "golang"
}

func (s *Scanner) RequiredExecutable() string {
	return "govulncheck"
}

// Detect is retained for compatibility with filesystem-based tests.
func (s *Scanner) Detect(source depscan.Source) depscan.Detection {
	manifestPath := filepath.Join(source.Directory, "go.mod")

	if _, err := os.Stat(manifestPath); err != nil {
		return depscan.Detection{}
	}

	return depscan.Detection{
		Present:      true,
		Supported:    true,
		ManifestFile: source.ManifestPath("go.mod"),
		LockFile:     "go.mod",
	}
}

// DetectWithExecutor checks go.mod inside runtime containers.
func (s *Scanner) DetectWithExecutor(
	ctx context.Context,
	source depscan.Source,
	executor depscan.Executor,
) (depscan.Detection, error) {
	var probeErrors []error

	for _, runtime := range source.Runtimes {
		exists, err := executor.FileExists(
			ctx,
			runtime,
			"go.mod",
		)

		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"check go.mod in %s: %w",
					runtime.ContainerName,
					err,
				),
			)
			continue
		}

		if exists {
			return depscan.Detection{
				Present:      true,
				Supported:    true,
				ManifestFile: source.ManifestPath("go.mod"),
				LockFile:     "go.mod",
			}, nil
		}
	}

	if len(probeErrors) > 0 {
		return depscan.Detection{}, errors.Join(probeErrors...)
	}

	return depscan.Detection{}, nil
}

func (s *Scanner) Scan(
	ctx context.Context,
	source depscan.Source,
	runtime depscan.Runtime,
	executor depscan.Executor,
	detection depscan.Detection,
) (depscan.EcosystemResult, error) {
	result := depscan.EcosystemResult{
		Ecosystem:    s.Ecosystem(),
		ManifestFile: detection.ManifestFile,
		Status:       depscan.StatusFailed,
		Findings:     make([]depscan.Finding, 0),
	}

	if !detection.Present || !detection.Supported {
		result.Status = depscan.StatusUnsupported
		result.ErrorMessage = "Go module is not supported"
		return result, nil
	}

	// govulncheck source analysis requires the Go toolchain.
	if err := executor.Probe(ctx, runtime, "go"); err != nil {
		result.ErrorMessage = fmt.Sprintf(
			"Go toolchain is unavailable: %v",
			err,
		)
		return result, nil
	}

	command, executionErr := executor.Execute(
		ctx,
		runtime,
		s.RequiredExecutable(),
		"-json",
		"./...",
	)

	if executionErr != nil {
		result.ErrorMessage = fmt.Sprintf(
			"govulncheck execution failed: %v",
			executionErr,
		)
		return result, nil
	}

	findings, err := parseFindings(command.Stdout)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf(
			"unable to parse govulncheck JSON: %v",
			err,
		)
		return result, nil
	}

	result.Status = depscan.StatusCompleted
	result.Findings = findings

	return result, nil
}
