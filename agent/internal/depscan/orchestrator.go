package depscan

import (
	"context"
	"errors"
	"fmt"
)

type Orchestrator struct {
	registry *Registry
	executor Executor
}

func NewOrchestrator(
	registry *Registry,
	executor Executor,
) (*Orchestrator, error) {
	if registry == nil {
		return nil, errors.New("scanner registry cannot be nil")
	}

	if executor == nil {
		return nil, errors.New("scanner executor cannot be nil")
	}

	return &Orchestrator{
		registry: registry,
		executor: executor,
	}, nil
}

// Scan executes every registered scanner detected in the source.
func (o *Orchestrator) Scan(
	ctx context.Context,
	source Source,
) ([]EcosystemResult, error) {
	if o == nil {
		return nil, errors.New("orchestrator is nil")
	}

	results := make([]EcosystemResult, 0)

	for _, scanner := range o.registry.All() {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		var detection Detection
		var detectionErr error

		if detector, ok := scanner.(ContainerDetector); ok {
			detection, detectionErr = detector.DetectWithExecutor(
				ctx,
				source,
				o.executor,
			)
		} else {
			// Backward compatibility for existing test scanners.
			detection = scanner.Detect(source)
		}

		if detectionErr != nil {
			results = append(results, EcosystemResult{
				Ecosystem:    scanner.Ecosystem(),
				ManifestFile: source.ManifestPath("source"),
				Status:       StatusFailed,
				ErrorMessage: fmt.Sprintf(
					"manifest detection failed: %v",
					detectionErr,
				),
				Findings: make([]Finding, 0),
			})

			continue
		}

		// This ecosystem does not exist in the source.
		if !detection.Present {
			continue
		}

		result := EcosystemResult{
			Ecosystem:    scanner.Ecosystem(),
			ManifestFile: detection.ManifestFile,
			Status:       StatusFailed,
			Findings:     make([]Finding, 0),
		}

		// A manifest exists, but the scanner cannot audit it
		// because a required lockfile or configuration is missing.
		if !detection.Supported {
			result.Status = StatusUnsupported
			result.ErrorMessage = detection.UnsupportedReason

			results = append(results, result)
			continue
		}

		runtime, err := o.selectRuntime(
			ctx,
			source.Runtimes,
			scanner.RequiredExecutable(),
		)

		if err != nil {
			result.ErrorMessage = err.Error()

			results = append(results, result)
			continue
		}

		scanResult, scanErr := scanner.Scan(
			ctx,
			source,
			runtime,
			o.executor,
			detection,
		)

		if scanErr != nil {
			result.ErrorMessage = scanErr.Error()

			results = append(results, result)
			continue
		}

		// Normalize scanner output.
		if scanResult.Ecosystem == "" {
			scanResult.Ecosystem = scanner.Ecosystem()
		}

		if scanResult.ManifestFile == "" {
			scanResult.ManifestFile = detection.ManifestFile
		}

		if scanResult.Findings == nil {
			scanResult.Findings = make([]Finding, 0)
		}

		if scanResult.Status == "" {
			scanResult.Status = StatusFailed
			scanResult.ErrorMessage = "scanner returned an empty status"
		}

		results = append(results, scanResult)
	}

	return results, nil
}

// selectRuntime finds the first runtime that provides
// the executable required by the ecosystem scanner.
func (o *Orchestrator) selectRuntime(
	ctx context.Context,
	runtimes []Runtime,
	executable string,
) (Runtime, error) {
	if executable == "" {
		return Runtime{}, errors.New(
			"scanner required executable cannot be empty",
		)
	}

	if len(runtimes) == 0 {
		return Runtime{}, fmt.Errorf(
			"no runtime candidates available for executable %q",
			executable,
		)
	}

	var probeErrors []error

	for _, runtime := range runtimes {
		if err := ctx.Err(); err != nil {
			return Runtime{}, err
		}

		err := o.executor.Probe(
			ctx,
			runtime,
			executable,
		)

		if err == nil {
			return runtime, nil
		}

		probeErrors = append(
			probeErrors,
			fmt.Errorf(
				"service %q: %w",
				runtime.ServiceName,
				err,
			),
		)
	}

	return Runtime{}, fmt.Errorf(
		"no runtime provides executable %q: %w",
		executable,
		errors.Join(probeErrors...),
	)
}
