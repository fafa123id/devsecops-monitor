package python

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
	return "pip"
}

func (s *Scanner) RequiredExecutable() string {
	return "pip-audit"
}

// Detect is retained for compatibility with local filesystem tests.
func (s *Scanner) Detect(source depscan.Source) depscan.Detection {
	manifest := filepath.Join(source.Directory, "requirements.txt")

	info, err := os.Stat(manifest)
	if err != nil || info.IsDir() {
		return depscan.Detection{}
	}

	return depscan.Detection{
		Present:      true,
		Supported:    true,
		ManifestFile: source.ManifestPath("requirements.txt"),
		LockFile:     "requirements.txt",
	}
}

// DetectWithExecutor checks the manifest inside the target container.
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
			"requirements.txt",
		)

		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"check requirements.txt in %s: %w",
					runtime.ContainerName,
					err,
				),
			)
			continue
		}

		if !exists {
			continue
		}

		return depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: source.ManifestPath("requirements.txt"),
			LockFile:     "requirements.txt",
		}, nil
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

	if !detection.Present {
		result.Status = depscan.StatusUnsupported
		result.ErrorMessage = "requirements.txt not found"
		return result, nil
	}

	command, executionErr := executor.Execute(
		ctx,
		runtime,
		s.RequiredExecutable(),
		"--requirement",
		detection.LockFile,
		"--format=json",
	)

	// pip-audit returns exit code 1 when vulnerabilities are found.
	if executionErr != nil && command.ExitCode != 1 {
		result.ErrorMessage = fmt.Sprintf(
			"pip-audit execution failed: %v",
			executionErr,
		)
		return result, nil
	}

	if command.ExitCode != 0 && command.ExitCode != 1 {
		result.ErrorMessage = fmt.Sprintf(
			"pip-audit returned exit code %d: %s",
			command.ExitCode,
			string(command.Stderr),
		)
		return result, nil
	}

	findings, err := parseFindingsFromContent(
		command.Stdout,
		detection.ManifestFile,
	)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf(
			"unable to parse pip-audit JSON: %v",
			err,
		)
		return result, nil
	}

	result.Status = depscan.StatusCompleted
	result.Findings = findings

	return result, nil
}
