package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"devsecops-agent/internal/depscan"
)

// ContainerDetector performs manifest detection through
// a runtime container instead of the agent filesystem.

type Scanner struct{}

func New() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Ecosystem() string {
	return "composer"
}

func (s *Scanner) RequiredExecutable() string {
	return "composer"
}

func (s *Scanner) Detect(
	source depscan.Source,
) depscan.Detection {
	manifestPath := filepath.Join(
		source.Directory,
		"composer.json",
	)

	lockPath := filepath.Join(
		source.Directory,
		"composer.lock",
	)

	if _, err := os.Stat(manifestPath); err != nil {
		return depscan.Detection{}
	}

	if _, err := os.Stat(lockPath); err != nil {
		return depscan.Detection{
			Present:           true,
			Supported:         false,
			ManifestFile:      source.ManifestPath("composer.json"),
			UnsupportedReason: "composer.lock not found",
		}
	}

	return depscan.Detection{
		Present:      true,
		Supported:    true,
		ManifestFile: source.ManifestPath("composer.lock"),
		LockFile:     "composer.lock",
	}
}
func (s *Scanner) DetectWithExecutor(
	ctx context.Context,
	source depscan.Source,
	executor depscan.Executor,
) (depscan.Detection, error) {
	var probeErrors []error

	for _, runtime := range source.Runtimes {
		manifestExists, err := executor.FileExists(
			ctx,
			runtime,
			"composer.json",
		)

		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"check composer.json in %s: %w",
					runtime.ContainerName,
					err,
				),
			)
			continue
		}

		if !manifestExists {
			continue
		}

		lockExists, err := executor.FileExists(
			ctx,
			runtime,
			"composer.lock",
		)

		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"check composer.lock in %s: %w",
					runtime.ContainerName,
					err,
				),
			)
			continue
		}

		if !lockExists {
			return depscan.Detection{
				Present:           true,
				Supported:         false,
				ManifestFile:      source.ManifestPath("composer.json"),
				UnsupportedReason: "composer.lock not found",
			}, nil
		}

		return depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: source.ManifestPath("composer.lock"),
			LockFile:     "composer.lock",
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
		result.ErrorMessage = "composer.json not found"
		return result, nil
	}

	if !detection.Supported {
		result.Status = depscan.StatusUnsupported
		result.ErrorMessage = detection.UnsupportedReason
		return result, nil
	}

	command, executionErr := executor.Execute(
		ctx,
		runtime,
		s.RequiredExecutable(),
		"audit",
		"--format=json",
		"--locked",
	)

	// Composer uses exit code 1 when security advisories
	// or dependency policy issues are found.
	// We must still parse its JSON output.
	if executionErr != nil && command.ExitCode != 1 {
		result.ErrorMessage = fmt.Sprintf(
			"composer audit execution failed: %v",
			executionErr,
		)

		return result, nil
	}

	if command.ExitCode != 0 && command.ExitCode != 1 {
		result.ErrorMessage = fmt.Sprintf(
			"composer audit returned exit code %d",
			command.ExitCode,
		)

		return result, nil
	}

	var report auditReport

	if err := json.Unmarshal(command.Stdout, &report); err != nil {
		result.ErrorMessage = fmt.Sprintf(
			"unable to parse Composer audit JSON: %v",
			err,
		)

		return result, nil
	}

	lockContent, err := executor.ReadFile(
		ctx,
		runtime,
		detection.LockFile,
	)

	if err != nil {
		result.ErrorMessage = fmt.Sprintf(
			"unable to read Composer lockfile: %v",
			err,
		)
		return result, nil
	}

	result.Status = depscan.StatusCompleted
	result.Findings = parseFindingsFromContent(
		report,
		lockContent,
		detection.ManifestFile,
	)

	return result, nil
}
