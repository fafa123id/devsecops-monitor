package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/moby/moby/client"
)

const (
	composeProjectLabel     = "com.docker.compose.project"
	composeServiceLabel     = "com.docker.compose.service"
	composeWorkingDirLabel  = "com.docker.compose.project.working_dir"
	composeConfigFilesLabel = "com.docker.compose.project.config_files"
)

type ProjectInventory struct {
	DockerProjectName string               `json:"docker_project_name"`
	SourceDirectory   string               `json:"source_directory,omitempty"`
	ComposeFiles      []string             `json:"compose_files,omitempty"`
	Containers        []ContainerInventory `json:"containers"`
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
	sourceDirectories := make(map[string]string)
	composeFiles := make(map[string][]string)
	for _, item := range result.Items {
		projectName := item.Labels[composeProjectLabel]

		// Skip containers that are not managed by Docker Compose.
		if projectName == "" {
			continue
		}
		workingDir := strings.TrimSpace(
			item.Labels[composeWorkingDirLabel],
		)

		configFiles := strings.TrimSpace(
			item.Labels[composeConfigFilesLabel],
		)

		if workingDir != "" {
			existingDir := sourceDirectories[projectName]

			if existingDir == "" {
				sourceDirectories[projectName] = workingDir
			} else if !strings.EqualFold(existingDir, workingDir) {
				return nil, fmt.Errorf(
					"inconsistent working directories for compose project %q: %q and %q",
					projectName,
					existingDir,
					workingDir,
				)
			}
		}

		if configFiles != "" {
			for _, file := range strings.Split(configFiles, ",") {
				file = strings.TrimSpace(file)

				if file != "" {
					composeFiles[projectName] = append(
						composeFiles[projectName],
						file,
					)
				}
			}
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
		files := composeFiles[name]
		sort.Strings(files)

		inventory = append(inventory, ProjectInventory{
			DockerProjectName: name,
			SourceDirectory:   sourceDirectories[name],
			ComposeFiles:      files,
			Containers:        containers,
		})
	}

	sort.Slice(inventory, func(i, j int) bool {
		return inventory[i].DockerProjectName < inventory[j].DockerProjectName
	})

	return inventory, nil
}
func resolveImageRepoDigest(
	ctx context.Context,
	dockerClient *client.Client,
	imageID string,
) (string, error) {
	if strings.TrimSpace(imageID) == "" {
		return "", fmt.Errorf("image ID cannot be empty")
	}

	result, err := dockerClient.ImageInspect(ctx, imageID)
	if err != nil {
		return "", fmt.Errorf("inspect image %s: %w", imageID, err)
	}

	repoDigests := result.RepoDigests

	if len(repoDigests) == 0 {
		return "", nil
	}

	// Sort to make the selected digest deterministic.
	sort.Strings(repoDigests)

	return repoDigests[0], nil
}
