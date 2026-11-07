package npm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"devsecops-agent/internal/depscan"
)

type Scanner struct{}

func New() *Scanner {
	return &Scanner{}
}

func (s *Scanner) Ecosystem() string {
	return "npm"
}

func (s *Scanner) RequiredExecutable() string {
	return "npm"
}

func (s *Scanner) Detect(
	source depscan.Source,
) depscan.Detection {
	packageJSON := filepath.Join(
		source.Directory,
		"package.json",
	)

	if !fileExists(packageJSON) {
		return depscan.Detection{}
	}

	lockfile := lockfileName(source.Directory)

	if lockfile == "" {
		return depscan.Detection{
			Present:           true,
			Supported:         false,
			ManifestFile:      source.ManifestPath("package.json"),
			UnsupportedReason: "package.json found, but no npm lockfile (package-lock.json or npm-shrinkwrap.json) is available",
		}
	}

	return depscan.Detection{
		Present:      true,
		Supported:    true,
		ManifestFile: source.ManifestPath(lockfile),
		LockFile:     lockfile,
	}
}
func (s *Scanner) DetectWithExecutor(
	ctx context.Context,
	source depscan.Source,
	executor depscan.Executor,
) (depscan.Detection, error) {
	var probeErrors []error

	for _, runtime := range source.Runtimes {
		packageJSONExists, err := executor.FileExists(
			ctx,
			runtime,
			"package.json",
		)

		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"check package.json in %s: %w",
					runtime.ContainerName,
					err,
				),
			)
			continue
		}

		if !packageJSONExists {
			continue
		}

		shrinkwrapExists, err := executor.FileExists(
			ctx,
			runtime,
			"npm-shrinkwrap.json",
		)

		if err != nil {
			probeErrors = append(
				probeErrors,
				fmt.Errorf(
					"check npm-shrinkwrap.json in %s: %w",
					runtime.ContainerName,
					err,
				),
			)
			continue
		}

		lockfile := ""

		if shrinkwrapExists {
			lockfile = "npm-shrinkwrap.json"
		} else {
			packageLockExists, err := executor.FileExists(
				ctx,
				runtime,
				"package-lock.json",
			)

			if err != nil {
				probeErrors = append(
					probeErrors,
					fmt.Errorf(
						"check package-lock.json in %s: %w",
						runtime.ContainerName,
						err,
					),
				)
				continue
			}

			if packageLockExists {
				lockfile = "package-lock.json"
			}
		}

		if lockfile == "" {
			return depscan.Detection{
				Present:           true,
				Supported:         false,
				ManifestFile:      source.ManifestPath("package.json"),
				UnsupportedReason: "package.json found, but no npm lockfile (package-lock.json or npm-shrinkwrap.json) is available",
			}, nil
		}

		return depscan.Detection{
			Present:      true,
			Supported:    true,
			ManifestFile: source.ManifestPath(lockfile),
			LockFile:     lockfile,
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
		result.ErrorMessage = "package.json not found"
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
		"--json",
	)

	// npm audit returns exit code 1 when vulnerabilities
	// are found. Its JSON output is still valid.
	if executionErr != nil && command.ExitCode != 1 {
		result.ErrorMessage = fmt.Sprintf(
			"npm audit execution failed: %v",
			executionErr,
		)

		return result, nil
	}

	if command.ExitCode != 0 && command.ExitCode != 1 {
		result.ErrorMessage = fmt.Sprintf(
			"npm audit returned exit code %d",
			command.ExitCode,
		)

		return result, nil
	}

	var report auditReport

	if err := json.Unmarshal(command.Stdout, &report); err != nil {
		result.ErrorMessage = fmt.Sprintf(
			"unable to parse npm audit JSON: %v",
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
			"unable to read npm lockfile: %v",
			err,
		)
		return result, nil
	}

	findings, err := parseFindingsFromContent(
		report,
		lockContent,
		detection.ManifestFile,
	)
	if err != nil {
		result.ErrorMessage = err.Error()
		return result, nil
	}

	result.Status = depscan.StatusCompleted
	result.Findings = findings

	return result, nil
}
