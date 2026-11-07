package main

import "testing"

func TestFilterScannableProjectsOnlyReturnsRunningProjects(t *testing.T) {
	inventory := []ProjectInventory{
		{
			DockerProjectName: "active-project",
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
			DockerProjectName: "paused-project",
			Containers: []ContainerInventory{
				{ContainerState: "paused"},
			},
		},
	}

	got := filterScannableProjects(inventory)

	if len(got) != 1 {
		t.Fatalf("expected 1 scannable project, got %d", len(got))
	}

	if got[0].DockerProjectName != "active-project" {
		t.Fatalf(
			"expected active-project, got %s",
			got[0].DockerProjectName,
		)
	}
}

func TestReconcileKnownProjectsRemovesInactiveAndMissingProjects(t *testing.T) {
	known := map[string]struct{}{
		"active-project":  {},
		"stopped-project": {},
		"missing-project": {},
	}

	inventory := []ProjectInventory{
		{
			DockerProjectName: "active-project",
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
	}

	reconcileKnownProjects(inventory, known)

	if len(known) != 1 {
		t.Fatalf("expected 1 known project, got %d", len(known))
	}

	if _, exists := known["active-project"]; !exists {
		t.Fatal("active project should remain known")
	}

	if _, exists := known["stopped-project"]; exists {
		t.Fatal("stopped project should be removed from known projects")
	}

	if _, exists := known["missing-project"]; exists {
		t.Fatal("missing project should be removed from known projects")
	}
}

func TestFindNewProjectsIgnoresStoppedProjects(t *testing.T) {
	inventory := []ProjectInventory{
		{
			DockerProjectName: "new-active",
			Containers: []ContainerInventory{
				{ContainerState: "running"},
			},
		},
		{
			DockerProjectName: "new-stopped",
			Containers: []ContainerInventory{
				{ContainerState: "exited"},
			},
		},
	}

	got := findNewProjects(
		inventory,
		make(map[string]struct{}),
	)

	if len(got) != 1 {
		t.Fatalf("expected 1 new project, got %d", len(got))
	}

	if got[0].DockerProjectName != "new-active" {
		t.Fatalf("expected new-active, got %s", got[0].DockerProjectName)
	}
}
