package main

import "testing"

func TestFilterPolicyEligibleProjects(t *testing.T) {
	inventory := []ProjectInventory{
		{
			DockerProjectName: "active-project",
			Containers: []ContainerInventory{
				{ContainerState: "running"},
			},
		},
		{
			DockerProjectName: "archived-project",
			Containers: []ContainerInventory{
				{ContainerState: "running"},
			},
		},
		{
			DockerProjectName: "disabled-project",
			Containers: []ContainerInventory{
				{ContainerState: "running"},
			},
		},
		{
			DockerProjectName: "weekly-disabled-project",
			Containers: []ContainerInventory{
				{ContainerState: "running"},
			},
		},
		{
			DockerProjectName: "stopped-project",
			Containers: []ContainerInventory{
				{ContainerState: "exited"},
			},
		},
		{
			DockerProjectName: "unknown-project",
			Containers: []ContainerInventory{
				{ContainerState: "running"},
			},
		},
	}

	policies := map[string]ProjectPolicy{
		"active-project": {
			Status:            "active",
			ScanSource:        true,
			WeeklyScanEnabled: true,
		},
		"archived-project": {
			Status:            "archived",
			ScanSource:        true,
			WeeklyScanEnabled: true,
		},
		"disabled-project": {
			Status:     "active",
			ScanSource: false,
		},
		"weekly-disabled-project": {
			Status:            "active",
			ScanSource:        true,
			WeeklyScanEnabled: false,
		},
		"stopped-project": {
			Status:     "stopped",
			ScanSource: true,
		},
	}

	// Normal scanning: weekly_scan_enabled is not required.
	normal := filterPolicyEligibleProjects(
		inventory,
		policies,
		false,
	)

	if len(normal) != 2 {
		t.Fatalf("expected 2 normal scan projects, got %d", len(normal))
	}

	// Weekly scanning: weekly_scan_enabled must also be true.
	weekly := filterPolicyEligibleProjects(
		inventory,
		policies,
		true,
	)

	if len(weekly) != 1 {
		t.Fatalf("expected 1 weekly scan project, got %d", len(weekly))
	}

	if weekly[0].DockerProjectName != "active-project" {
		t.Fatalf(
			"expected active-project, got %s",
			weekly[0].DockerProjectName,
		)
	}
}
