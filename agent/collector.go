package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/moby/moby/client"
)

const (
	composeProjectLabel = "com.docker.compose.project"
	composeServiceLabel = "com.docker.compose.service"
)

type ProjectInventory struct {
	DockerProjectName string               `json:"docker_project_name"`
	Containers        []ContainerInventory `json:"containers"`
}

type ImageScanTarget struct {
	ImageReference string
	ImageID        string
	ImageDigest    string
}

type ContainerInventory struct {
	ContainerID    string `json:"container_id"`
	ContainerName  string `json:"container_name"`
	ServiceName    string `json:"service_name,omitempty"`
	ImageReference string `json:"image_reference,omitempty"`
	ImageID        string `json:"image_id,omitempty"`
	ImageDigest    string `json:"image_digest,omitempty"`
	ContainerState string `json:"container_state"`
}

func collectInventory(
	ctx context.Context,
	dockerClient *client.Client,
) ([]ProjectInventory, error) {
	result, err := dockerClient.ContainerList(
		ctx,
		client.ContainerListOptions{
			All: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	projects := make(map[string][]ContainerInventory)

	for _, item := range result.Items {
		projectName := item.Labels[composeProjectLabel]

		// Skip containers that are not managed by Docker Compose.
		if projectName == "" {
			continue
		}

		containerName := ""
		if len(item.Names) > 0 {
			containerName = strings.TrimPrefix(item.Names[0], "/")
		}

		imageDigest := ""
		if item.ImageManifestDescriptor != nil {
			imageDigest = string(item.ImageManifestDescriptor.Digest)
		}

		container := ContainerInventory{
			ContainerID:    item.ID,
			ContainerName:  containerName,
			ServiceName:    item.Labels[composeServiceLabel],
			ImageReference: item.Image,
			ImageID:        item.ImageID,
			ImageDigest:    imageDigest,
			ContainerState: fmt.Sprint(item.State),
		}

		projects[projectName] = append(
			projects[projectName],
			container,
		)
	}

	inventory := make([]ProjectInventory, 0, len(projects))

	for name, containers := range projects {
		inventory = append(inventory, ProjectInventory{
			DockerProjectName: name,
			Containers:        containers,
		})
	}

	sort.Slice(inventory, func(i, j int) bool {
		return inventory[i].DockerProjectName < inventory[j].DockerProjectName
	})

	return inventory, nil
}
func collectUniqueImages(inventory []ProjectInventory) []ImageScanTarget {
	uniqueImages := make(map[string]ImageScanTarget)

	for _, project := range inventory {
		for _, container := range project.Containers {
			image := ImageScanTarget{
				ImageReference: container.ImageReference,
				ImageID:        container.ImageID,
				ImageDigest:    container.ImageDigest,
			}

			// Prefer immutable image identity.
			key := image.ImageID

			if key == "" {
				key = image.ImageDigest
			}

			if key == "" {
				key = image.ImageReference
			}

			if key == "" {
				continue
			}

			uniqueImages[key] = image
		}
	}

	images := make([]ImageScanTarget, 0, len(uniqueImages))

	for _, image := range uniqueImages {
		images = append(images, image)
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].ImageReference < images[j].ImageReference
	})

	return images
}
