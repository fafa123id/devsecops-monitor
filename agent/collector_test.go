package main

import "testing"

func TestCollectUniqueImages(t *testing.T) {
	inventory := []ProjectInventory{
		{
			DockerProjectName: "deployment",
			Containers: []ContainerInventory{
				{
					ContainerName:  "deployment-api-1",
					ImageReference: "deployment-api:latest",
					ImageID:        "sha256:aaa",
				},
				{
					ContainerName:  "deployment-worker-1",
					ImageReference: "deployment-api:latest",
					ImageID:        "sha256:aaa",
				},
				{
					ContainerName:  "deployment-redis-1",
					ImageReference: "redis:7-alpine",
					ImageID:        "sha256:bbb",
				},
			},
		},
	}

	images := collectUniqueImages(inventory)

	if len(images) != 2 {
		t.Fatalf("expected 2 unique images, got %d", len(images))
	}
}
