package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
)

type sourceMount struct {
	HostPath      string
	ContainerPath string
	ContainerID   string
	ContainerName string
	ServiceName   string
}

type dockerInspectContainer struct {
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`

	Mounts []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func discoverSourceMounts(
	ctx context.Context,
	project ProjectInventory,
) ([]sourceMount, error) {
	unique := make(map[string]sourceMount)

	for _, container := range project.Containers {
		if container.ContainerID == "" {
			continue
		}

		output, err := exec.CommandContext(
			ctx,
			"docker",
			"inspect",
			container.ContainerID,
		).Output()

		if err != nil {
			log.Printf(
				"Unable to inspect container %s: %v",
				container.ContainerName,
				err,
			)
			continue
		}

		var inspected []dockerInspectContainer

		if err := json.Unmarshal(output, &inspected); err != nil {
			log.Printf(
				"Unable to parse Docker inspect result for %s: %v",
				container.ContainerName,
				err,
			)
			continue
		}

		if len(inspected) == 0 || !inspected[0].State.Running {
			continue
		}

		for _, mount := range inspected[0].Mounts {
			if mount.Type != "bind" {
				continue
			}

			// Skip Docker socket mounts.
			if strings.HasSuffix(strings.ToLower(mount.Source), ".sock") ||
				strings.HasSuffix(strings.ToLower(mount.Destination), ".sock") {
				continue
			}

			if mount.Source == "" || mount.Destination == "" {
				continue
			}

			hostPath := filepath.Clean(mount.Source)

			keyHostPath := hostPath

			if goruntime.GOOS == "windows" {
				keyHostPath = strings.ToLower(keyHostPath)
			}

			key := strings.Join([]string{
				keyHostPath,
				filepath.Clean(mount.Destination),
				container.ContainerID,
			}, "|")

			unique[key] = sourceMount{
				HostPath:      hostPath,
				ContainerPath: mount.Destination,
				ContainerID:   container.ContainerID,
				ContainerName: container.ContainerName,
				ServiceName:   container.ServiceName,
			}
		}
	}

	result := make([]sourceMount, 0, len(unique))

	for _, mount := range unique {
		result = append(result, mount)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].HostPath != result[j].HostPath {
			return result[i].HostPath < result[j].HostPath
		}

		if result[i].ServiceName != result[j].ServiceName {
			return result[i].ServiceName < result[j].ServiceName
		}

		return result[i].ContainerName < result[j].ContainerName
	})

	return result, nil
}
