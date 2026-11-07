package main

import (
	"context"
	"devsecops-agent/internal/depscan"
	composerscanner "devsecops-agent/internal/depscan/composer"
	golangscanner "devsecops-agent/internal/depscan/golang"
	npmscanner "devsecops-agent/internal/depscan/npm"
	pythonscanner "devsecops-agent/internal/depscan/python"
	"fmt"
	"log"
	"sort"
	"time"
)

// newDependencyOrchestrator registers all supported ecosystem scanners.
// New ecosystems should be registered here, not hardcoded in scanProject.
func newDependencyOrchestrator() (*depscan.Orchestrator, error) {
	registry := depscan.NewRegistry()

	if err := registry.Register(composerscanner.New()); err != nil {
		return nil, fmt.Errorf(
			"register Composer scanner: %w",
			err,
		)
	}
	if err := registry.Register(npmscanner.New()); err != nil {
		return nil, fmt.Errorf(
			"register npm scanner: %w",
			err,
		)
	}
	if err := registry.Register(golangscanner.New()); err != nil {
		return nil, fmt.Errorf(
			"register Go scanner: %w",
			err,
		)
	}
	if err := registry.Register(pythonscanner.New()); err != nil {
		return nil, fmt.Errorf(
			"register Python scanner: %w",
			err,
		)
	}
	executor := depscan.NewDockerExecutor()

	orchestrator, err := depscan.NewOrchestrator(
		registry,
		executor,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"initialize dependency orchestrator: %w",
			err,
		)
	}

	return orchestrator, nil
}

func scanProject(
	ctx context.Context,
	orchestrator *depscan.Orchestrator,
	project ProjectInventory,
) ProjectScanPayload {
	payload := ProjectScanPayload{
		DockerProjectName: project.DockerProjectName,
		Ecosystems:        make([]EcosystemScanPayload, 0),
	}

	sources, err := discoverProjectSources(ctx, project)
	if err != nil {
		payload.Ecosystems = append(
			payload.Ecosystems,
			depscan.EcosystemResult{
				Ecosystem:    "unknown",
				ManifestFile: "source",
				Status:       depscan.StatusFailed,
				ErrorMessage: fmt.Sprintf(
					"source discovery failed: %v",
					err,
				),
				Findings: make([]depscan.Finding, 0),
			},
		)

		return payload
	}

	if len(sources) == 0 {
		payload.Ecosystems = append(
			payload.Ecosystems,
			depscan.EcosystemResult{
				Ecosystem:    "unknown",
				ManifestFile: "source",
				Status:       depscan.StatusUnsupported,
				ErrorMessage: "no accessible source bind mount was found",
				Findings:     make([]depscan.Finding, 0),
			},
		)

		return payload
	}

	for _, source := range sources {
		fmt.Printf(
			"\nSource directory: %s\n",
			source.Directory,
		)

		fmt.Printf(
			"Runtime candidates: %d\n",
			len(source.Runtimes),
		)

		scanCtx, cancel := context.WithTimeout(
			ctx,
			15*time.Minute,
		)

		results, scanErr := orchestrator.Scan(
			scanCtx,
			source,
		)

		cancel()

		payload.Ecosystems = append(
			payload.Ecosystems,
			results...,
		)

		if scanErr != nil {
			log.Printf(
				"Dependency scan error for %s: %v",
				source.Directory,
				scanErr,
			)

			payload.Ecosystems = append(
				payload.Ecosystems,
				depscan.EcosystemResult{
					Ecosystem:    "unknown",
					ManifestFile: source.ManifestPath("source"),
					Status:       depscan.StatusFailed,
					ErrorMessage: scanErr.Error(),
					Findings:     make([]depscan.Finding, 0),
				},
			)
		}

		for _, result := range results {
			fmt.Printf(
				"  %s: %s | Findings: %d\n",
				result.Ecosystem,
				result.Status,
				len(result.Findings),
			)

			if result.ErrorMessage != "" {
				log.Printf(
					"%s scanner warning: %s",
					result.Ecosystem,
					result.ErrorMessage,
				)
			}
		}
	}

	// Ensure that Laravel always receives at least one
	// ecosystem result for each scanned Compose project.
	if len(payload.Ecosystems) == 0 {
		payload.Ecosystems = append(
			payload.Ecosystems,
			depscan.EcosystemResult{
				Ecosystem:    "unknown",
				ManifestFile: "source",
				Status:       depscan.StatusUnsupported,
				ErrorMessage: "no registered scanner detected a supported dependency manifest",
				Findings:     make([]depscan.Finding, 0),
			},
		)
	}

	sort.Slice(
		payload.Ecosystems,
		func(i, j int) bool {
			left := payload.Ecosystems[i]
			right := payload.Ecosystems[j]

			if left.Ecosystem == right.Ecosystem {
				return left.ManifestFile < right.ManifestFile
			}

			return left.Ecosystem < right.Ecosystem
		},
	)

	return payload
}

func scanInventory(
	ctx context.Context,
	inventory []ProjectInventory,
	triggerType string,
) (ScanSubmission, error) {
	submission := ScanSubmission{
		TriggerType:    triggerType,
		ScannerVersion: "dependency-audit-v1.2",
		Projects:       make([]ProjectScanPayload, 0),
	}

	orchestrator, err := newDependencyOrchestrator()
	if err != nil {
		return submission, err
	}

	for _, project := range inventory {
		if !projectHasRunningContainer(project) {
			continue
		}

		fmt.Printf(
			"\n========================================\n",
		)

		fmt.Printf(
			"Scanning Compose project: %s\n",
			project.DockerProjectName,
		)

		projectPayload := scanProject(
			ctx,
			orchestrator,
			project,
		)

		totalFindings := 0

		for _, ecosystem := range projectPayload.Ecosystems {
			totalFindings += len(ecosystem.Findings)
		}

		fmt.Printf(
			"\nProject scan completed: %s\n",
			project.DockerProjectName,
		)

		fmt.Printf(
			"Ecosystems: %d | Findings: %d\n",
			len(projectPayload.Ecosystems),
			totalFindings,
		)

		submission.Projects = append(
			submission.Projects,
			projectPayload,
		)
	}

	return submission, nil
}
