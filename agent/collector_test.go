package main

import (
	"context"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

func TestCollectInventory(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		t.Skipf("Docker client unavailable: %v", err)
	}
	defer dockerClient.Close()

	// This is an integration test. Skip when Docker Engine is unavailable.
	if _, err := dockerClient.Info(ctx, client.InfoOptions{}); err != nil {
		t.Skipf("Docker Engine unavailable: %v", err)
	}

	inventory, err := collectInventory(ctx, dockerClient)
	if err != nil {
		t.Fatalf("collect inventory: %v", err)
	}

	seenProjects := make(map[string]bool)
	previousProject := ""

	totalContainers := 0

	for _, project := range inventory {
		if project.DockerProjectName == "" {
			t.Error("Docker project name must not be empty")
			continue
		}

		if seenProjects[project.DockerProjectName] {
			t.Errorf(
				"duplicate Compose project: %s",
				project.DockerProjectName,
			)
		}

		seenProjects[project.DockerProjectName] = true

		if previousProject > project.DockerProjectName {
			t.Errorf(
				"projects are not sorted: %s appears before %s",
				previousProject,
				project.DockerProjectName,
			)
		}

		previousProject = project.DockerProjectName

		for _, container := range project.Containers {
			if container.ContainerID == "" {
				t.Errorf(
					"container ID is empty in project %s",
					project.DockerProjectName,
				)
			}

			totalContainers++
		}

		t.Logf(
			"Project: %s | Source: %s | Containers: %d",
			project.DockerProjectName,
			project.SourceDirectory,
			len(project.Containers),
		)
	}

	t.Logf("Projects discovered: %d", len(inventory))
	t.Logf("Containers discovered: %d", totalContainers)
}
