package main

import (
	"testing"
	"time"
)

func testWIB(
	year int,
	month time.Month,
	day, hour, minute int,
) time.Time {
	return time.Date(
		year,
		month,
		day,
		hour,
		minute,
		0,
		0,
		weeklyScanLocation,
	)
}

func TestFindNewProjects(t *testing.T) {
	inventory := []ProjectInventory{
		{
			DockerProjectName: "deployment",
			Containers: []ContainerInventory{
				{
					ContainerName:  "deployment-api-1",
					ContainerState: "running",
				},
			},
		},
		{
			DockerProjectName: "digisejahtera",
			Containers: []ContainerInventory{
				{
					ContainerName:  "digisejahtera-app-1",
					ContainerState: "running",
				},
			},
		},
		{
			DockerProjectName: "digisejahtera",
			Containers: []ContainerInventory{
				{
					ContainerName:  "digisejahtera-vite-1",
					ContainerState: "running",
				},
			},
		},
	}

	known := map[string]struct{}{
		"deployment": {},
	}

	result := findNewProjects(inventory, known)

	if len(result) != 1 {
		t.Fatalf(
			"expected 1 new project, got %d",
			len(result),
		)
	}

	if result[0].DockerProjectName != "digisejahtera" {
		t.Errorf(
			"unexpected new project: %s",
			result[0].DockerProjectName,
		)
	}
}

func TestRememberProjects(t *testing.T) {
	known := make(map[string]struct{})

	projects := []ProjectInventory{
		{
			DockerProjectName: "deployment",
			Containers: []ContainerInventory{
				{ContainerName: "api"},
			},
		},
	}

	rememberProjects(projects, known)

	if _, exists := known["deployment"]; !exists {
		t.Fatal("project should be remembered")
	}
}

func TestWeeklyScanDue(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		lastScan string
		expected bool
	}{
		{
			name:     "before Monday schedule",
			now:      testWIB(2026, 11, 9, 1, 59),
			expected: false,
		},
		{
			name:     "Monday scheduled time",
			now:      testWIB(2026, 11, 9, 2, 0),
			expected: true,
		},
		{
			name:     "Monday already scanned",
			now:      testWIB(2026, 11, 9, 3, 0),
			lastScan: "2026-11-09",
			expected: false,
		},
		{
			name:     "catch up on Wednesday",
			now:      testWIB(2026, 11, 11, 10, 0),
			lastScan: "2026-11-02",
			expected: true,
		},
		{
			name:     "Sunday after this week's scheduled scan",
			now:      testWIB(2026, 11, 8, 23, 59),
			lastScan: "2026-11-02",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := weeklyScanDue(
				test.now,
				test.lastScan,
			)

			if actual != test.expected {
				t.Errorf(
					"expected %v, got %v",
					test.expected,
					actual,
				)
			}
		})
	}
}

func TestInitialWeeklyScanMarker(t *testing.T) {
	beforeSchedule := testWIB(2026, 11, 9, 1, 0)

	if marker := initialWeeklyScanMarker(beforeSchedule); marker != "" {
		t.Errorf("expected empty marker, got %s", marker)
	}

	afterSchedule := testWIB(2026, 11, 9, 3, 0)

	marker := initialWeeklyScanMarker(afterSchedule)

	if marker != "2026-11-09" {
		t.Errorf(
			"expected 2026-11-09, got %s",
			marker,
		)
	}
}

func TestSuccessfullyScannedProjects(t *testing.T) {
	targets := []ProjectInventory{
		{DockerProjectName: "deployment"},
		{DockerProjectName: "digisejahtera"},
	}

	submission := ScanSubmission{
		Projects: []ProjectScanPayload{
			{
				DockerProjectName: "deployment",
				Ecosystems: []EcosystemScanPayload{
					{Ecosystem: "composer", Status: "completed"},
					{Ecosystem: "npm", Status: "unsupported"},
				},
			},
			{
				DockerProjectName: "digisejahtera",
				Ecosystems: []EcosystemScanPayload{
					{Ecosystem: "composer", Status: "completed"},
					{Ecosystem: "npm", Status: "failed"},
				},
			},
		},
	}

	successful := successfullyScannedProjects(
		targets,
		submission,
	)

	if len(successful) != 1 {
		t.Fatalf(
			"expected 1 successful project, got %d",
			len(successful),
		)
	}

	if successful[0].DockerProjectName != "deployment" {
		t.Errorf(
			"unexpected successful project: %s",
			successful[0].DockerProjectName,
		)
	}
}
