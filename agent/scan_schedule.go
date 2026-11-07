package main

import (
	"strings"
	"time"
)

const weeklyScanHour = 2

var weeklyScanLocation = time.FixedZone(
	"WIB",
	7*60*60,
)

// projectKey generates a stable identity for a Compose project.
func projectKey(project ProjectInventory) string {
	return strings.ToLower(
		strings.TrimSpace(project.DockerProjectName),
	)
}

// projectHasRunningContainer reports whether a Compose project
// has at least one running container.
func projectHasRunningContainer(project ProjectInventory) bool {
	for _, container := range project.Containers {
		if strings.EqualFold(
			strings.TrimSpace(container.ContainerState),
			"running",
		) {
			return true
		}
	}

	return false
}

// filterScannableProjects returns only projects with running containers.
func filterScannableProjects(
	inventory []ProjectInventory,
) []ProjectInventory {
	result := make([]ProjectInventory, 0)

	for _, project := range inventory {
		if projectHasRunningContainer(project) {
			result = append(result, project)
		}
	}

	return result
}

// filterPolicyEligibleProjects returns running projects allowed
// by Laravel project policies.
//
// weeklyOnly determines whether weekly_scan_enabled must also be true.
func filterPolicyEligibleProjects(
	inventory []ProjectInventory,
	policies map[string]ProjectPolicy,
	weeklyOnly bool,
) []ProjectInventory {
	runningProjects := filterScannableProjects(inventory)

	result := make([]ProjectInventory, 0)

	for _, project := range runningProjects {
		key := policyProjectKey(project.DockerProjectName)

		policy, exists := policies[key]
		if !exists {
			// Fail closed when Laravel has no policy for a project.
			continue
		}

		if !policy.ScanSource {
			continue
		}

		// Only active or degraded projects can be scanned.
		if policy.Status != "active" && policy.Status != "degraded" {
			continue
		}

		if weeklyOnly && !policy.WeeklyScanEnabled {
			continue
		}

		result = append(result, project)
	}

	return result
}

// reconcileKnownProjects removes projects that are no longer active
// from the agent's in-memory scan tracking.
func reconcileKnownProjects(
	inventory []ProjectInventory,
	knownProjects map[string]struct{},
) {
	activeProjects := make(map[string]struct{})

	for _, project := range inventory {
		key := projectKey(project)

		if key != "" && projectHasRunningContainer(project) {
			activeProjects[key] = struct{}{}
		}
	}

	for key := range knownProjects {
		if _, active := activeProjects[key]; !active {
			delete(knownProjects, key)
		}
	}
}

// findNewProjects returns projects that have not been
// successfully scanned during this agent session.
func findNewProjects(
	inventory []ProjectInventory,
	knownProjects map[string]struct{},
) []ProjectInventory {
	result := make([]ProjectInventory, 0)
	discovered := make(map[string]struct{})

	for _, project := range inventory {
		key := projectKey(project)

		if key == "" || !projectHasRunningContainer(project) {
			continue
		}

		if _, exists := knownProjects[key]; exists {
			continue
		}

		// Avoid duplicate Compose project entries.
		if _, exists := discovered[key]; exists {
			continue
		}

		discovered[key] = struct{}{}
		result = append(result, project)
	}

	return result
}

// rememberProjects marks projects as successfully scanned.
// Call this only after Laravel accepts the scan submission.
func rememberProjects(
	projects []ProjectInventory,
	knownProjects map[string]struct{},
) {
	for _, project := range projects {
		key := projectKey(project)

		if key == "" {
			continue
		}

		knownProjects[key] = struct{}{}
	}
}

// weeklyScanKey returns the Monday date for the current WIB week.
func weeklyScanKey(now time.Time) string {
	local := now.In(weeklyScanLocation)

	daysSinceMonday := (int(local.Weekday()) + 6) % 7

	monday := time.Date(
		local.Year(),
		local.Month(),
		local.Day()-daysSinceMonday,
		0,
		0,
		0,
		0,
		weeklyScanLocation,
	)

	return monday.Format("2006-01-02")
}

// weeklyScanDue checks whether the weekly scheduled scan
// should run for the current WIB week.
//
// If the agent was offline on Monday, it performs a catch-up
// scan the next time the inventory loop runs.
func weeklyScanDue(
	now time.Time,
	lastWeeklyScanKey string,
) bool {
	local := now.In(weeklyScanLocation)

	key := weeklyScanKey(local)

	monday, err := time.ParseInLocation(
		"2006-01-02",
		key,
		weeklyScanLocation,
	)
	if err != nil {
		return false
	}

	scheduledAt := time.Date(
		monday.Year(),
		monday.Month(),
		monday.Day(),
		weeklyScanHour,
		0,
		0,
		0,
		weeklyScanLocation,
	)

	if local.Before(scheduledAt) {
		return false
	}

	return lastWeeklyScanKey != key
}

// initialWeeklyScanMarker determines whether the initial
// full scan already satisfies this week's scheduled scan.
//
// When the agent starts before Monday 02:00 WIB, the weekly
// scan remains scheduled for Monday at 02:00.
func initialWeeklyScanMarker(now time.Time) string {
	if !weeklyScanDue(now, "") {
		return ""
	}

	return weeklyScanKey(now)
}

func successfullyScannedProjects(
	targets []ProjectInventory,
	submission ScanSubmission,
) []ProjectInventory {
	successByProject := make(map[string]bool)

	for _, result := range submission.Projects {
		success := len(result.Ecosystems) > 0

		for _, ecosystem := range result.Ecosystems {
			if ecosystem.Status != "completed" &&
				ecosystem.Status != "unsupported" {
				success = false
				break
			}
		}

		successByProject[strings.ToLower(strings.TrimSpace(result.DockerProjectName))] = success
	}

	successful := make([]ProjectInventory, 0)

	for _, project := range targets {
		key := projectKey(project)

		if successByProject[key] {
			successful = append(successful, project)
		}
	}

	return successful
}
