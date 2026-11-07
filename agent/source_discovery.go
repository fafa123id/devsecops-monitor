package main

import (
	"context"

	"devsecops-agent/internal/depscan"
)

func discoverProjectSources(
	ctx context.Context,
	project ProjectInventory,
) ([]depscan.Source, error) {
	mounts, err := discoverSourceMounts(ctx, project)
	if err != nil {
		return nil, err
	}

	candidates := make(
		[]depscan.MountCandidate,
		0,
		len(mounts),
	)

	for _, mount := range mounts {
		candidates = append(
			candidates,
			depscan.MountCandidate{
				HostPath: mount.HostPath,
				Runtime: depscan.Runtime{
					ContainerID:   mount.ContainerID,
					ContainerName: mount.ContainerName,
					ServiceName:   mount.ServiceName,
					ContainerPath: mount.ContainerPath,
				},
			},
		)
	}

	return depscan.GroupSources(
		project.SourceDirectory,
		candidates,
	), nil
}
