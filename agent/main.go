package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/moby/moby/client"
)

func main() {
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer dockerClient.Close()

	result, err := dockerClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		log.Fatalf("Failed to connect to Docker Engine: %v", err)
	}

	info := result.Info

	fmt.Println("DevSecOps Monitor - Docker Agent")
	fmt.Println("--------------------------------")
	fmt.Printf("Docker Engine: %s\n", info.ServerVersion)
	fmt.Printf("Operating System: %s\n", info.OperatingSystem)
	fmt.Printf("Architecture: %s\n", info.Architecture)
	fmt.Printf("Containers: %d\n", info.Containers)
	fmt.Printf("Running Containers: %d\n", info.ContainersRunning)
	fmt.Printf("Images: %d\n", info.Images)

	fmt.Println("--------------------------------")
	fmt.Println("Collecting Docker inventory...")

	inventory, err := collectInventory(ctx, dockerClient)
	if err != nil {
		log.Fatalf("Failed to collect inventory: %v", err)
	}

	totalContainers := 0

	for _, project := range inventory {
		fmt.Printf("\nProject: %s\n", project.DockerProjectName)

		for _, container := range project.Containers {
			fmt.Printf("  Container: %s\n", container.ContainerName)
			fmt.Printf("    ID: %s\n", container.ContainerID[:12])
			fmt.Printf("    Service: %s\n", container.ServiceName)
			fmt.Printf("    Image: %s\n", container.ImageReference)
			fmt.Printf("    Digest: %s\n", container.ImageDigest)
			fmt.Printf("    State: %s\n", container.ContainerState)

			totalContainers++
		}
	}

	fmt.Println("\n--------------------------------")
	fmt.Printf("Projects discovered: %d\n", len(inventory))
	fmt.Printf("Containers discovered: %d\n", totalContainers)
	fmt.Println("Inventory collection successful.")
	fmt.Println("\nSynchronizing inventory with Laravel API...")

	if err := sendInventory(inventory); err != nil {
		log.Fatalf("Inventory synchronization failed: %v", err)
	}
	fmt.Println("\nAgent is running. Heartbeat interval: 30 seconds")

	heartbeatTicker := time.NewTicker(30 * time.Second)
	inventoryTicker := time.NewTicker(30 * time.Second)

	defer heartbeatTicker.Stop()
	defer inventoryTicker.Stop()

	fmt.Println("Heartbeat interval: 30 seconds")
	fmt.Println("Inventory sync interval: 30 seconds")

	for {
		select {
		case <-heartbeatTicker.C:
			if err := sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}

		case <-inventoryTicker.C:
			fmt.Println("\nRefreshing Docker inventory...")

			updatedInventory, err := collectInventory(ctx, dockerClient)
			if err != nil {
				log.Printf("Inventory collection failed: %v", err)
				continue
			}

			if err := sendInventory(updatedInventory); err != nil {
				log.Printf("Inventory synchronization failed: %v", err)
				continue
			}

			fmt.Println("Inventory refresh completed.")
		}
	}
}
