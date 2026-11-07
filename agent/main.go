package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/moby/moby/client"
)

const (
	heartbeatInterval = 10 * time.Second
	inventoryInterval = 10 * time.Second
	scanRetryDelay    = 1 * time.Minute
)

func main() {
	// -------------------------------------------------
	// Environment configuration
	// -------------------------------------------------

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Overload(); err != nil {
			log.Println("Warning: .env not loaded:", err)
		}
	}

	ctx := context.Background()

	// -------------------------------------------------
	// Docker client initialization
	// -------------------------------------------------

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

	// -------------------------------------------------
	// Initial inventory collection
	// -------------------------------------------------

	fmt.Println("\n--------------------------------")
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

			if len(container.ContainerID) >= 12 {
				fmt.Printf("    ID: %s\n", container.ContainerID[:12])
			} else {
				fmt.Printf("    ID: %s\n", container.ContainerID)
			}

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

	// -------------------------------------------------
	// Initial inventory synchronization
	// -------------------------------------------------

	fmt.Println("\nSynchronizing inventory with Laravel API...")

	if err := sendInventory(inventory); err != nil {
		log.Fatalf("Inventory synchronization failed: %v", err)
	}

	fmt.Println("Inventory synchronization completed.")

	// -------------------------------------------------
	// Scan scheduling state
	// -------------------------------------------------

	knownProjects := make(map[string]struct{})

	lastWeeklyScanKey := ""
	nextScanAttemptAt := time.Time{}

	// -------------------------------------------------
	// Initial policy synchronization
	// -------------------------------------------------

	fmt.Println("\nFetching project scanning policies...")

	policies, policyErr := fetchProjectPolicies()

	if policyErr != nil {
		log.Printf("Initial policy synchronization failed: %v", policyErr)
		log.Println("Vulnerability scanning will remain disabled until policies are available.")
	} else {
		scannableInventory := filterPolicyEligibleProjects(
			inventory,
			policies,
			false,
		)

		weeklyEligibleInventory := filterPolicyEligibleProjects(
			inventory,
			policies,
			true,
		)

		// -------------------------------------------------
		// Initial vulnerability scan
		// -------------------------------------------------

		if len(scannableInventory) == 0 {
			fmt.Println("No projects currently allowed for vulnerability scanning.")
		} else {
			fmt.Println("\nStarting initial vulnerability scan...")

			initialScanStartedAt := time.Now()

			scanCtx, cancelScan := context.WithTimeout(
				ctx,
				60*time.Minute,
			)
			submission, scanErr := scanInventory(
				scanCtx,
				scannableInventory,
				"discovery",
			)

			cancelScan()

			if scanErr != nil {
				log.Printf("Initial vulnerability scan failed: %v", scanErr)

				nextScanAttemptAt = time.Now().Add(scanRetryDelay)

			} else if len(submission.Projects) == 0 {
				log.Println("Initial scan generated no project results.")

				nextScanAttemptAt = time.Now().Add(scanRetryDelay)

			} else if err := sendScanResults(submission); err != nil {
				log.Printf(
					"Initial scan result submission failed: %v",
					err,
				)

				nextScanAttemptAt = time.Now().Add(scanRetryDelay)

			} else {
				successful := successfullyScannedProjects(
					scannableInventory,
					submission,
				)

				rememberProjects(
					successful,
					knownProjects,
				)

				// Initial scanning also satisfies this week's
				// weekly scan if all weekly-enabled projects
				// were successfully processed.
				successfulWeekly := successfullyScannedProjects(
					weeklyEligibleInventory,
					submission,
				)

				if len(successfulWeekly) == len(weeklyEligibleInventory) {
					lastWeeklyScanKey = initialWeeklyScanMarker(
						initialScanStartedAt,
					)
				}

				if len(successful) < len(scannableInventory) {
					nextScanAttemptAt = time.Now().Add(scanRetryDelay)

					log.Printf(
						"Some projects were not successfully scanned. Retry after %s.",
						scanRetryDelay,
					)
				}

				log.Printf(
					"Initial scan accepted: %d/%d projects successfully processed",
					len(successful),
					len(scannableInventory),
				)
			}
		}
	}

	// -------------------------------------------------
	// Periodic heartbeat and inventory synchronization
	// -------------------------------------------------

	heartbeatTicker := time.NewTicker(heartbeatInterval)
	inventoryTicker := time.NewTicker(inventoryInterval)

	defer heartbeatTicker.Stop()
	defer inventoryTicker.Stop()

	fmt.Println("\n--------------------------------")
	fmt.Println("Agent is running.")
	fmt.Printf("Heartbeat interval: %s\n", heartbeatInterval)
	fmt.Printf("Inventory sync interval: %s\n", inventoryInterval)
	fmt.Println("Project policy synchronization: enabled")
	fmt.Println("New and reactivated project scanning: enabled")
	fmt.Println("Archived projects: excluded")
	fmt.Println("Weekly scan: Monday 02:00 WIB")
	fmt.Printf("Failed scan retry: %s\n", scanRetryDelay)

	// -------------------------------------------------
	// Main agent loop
	// -------------------------------------------------

	for {
		select {
		// ---------------------------------------------
		// Heartbeat
		// ---------------------------------------------

		case <-heartbeatTicker.C:
			if err := sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}

		// ---------------------------------------------
		// Inventory synchronization
		// ---------------------------------------------

		case <-inventoryTicker.C:
			fmt.Println("\nRefreshing Docker inventory...")

			updatedInventory, err := collectInventory(
				ctx,
				dockerClient,
			)
			if err != nil {
				log.Printf("Inventory collection failed: %v", err)
				continue
			}

			// Always synchronize inventory first.
			if err := sendInventory(updatedInventory); err != nil {
				log.Printf(
					"Inventory synchronization failed: %v",
					err,
				)
				continue
			}

			fmt.Println("Inventory synchronization completed.")

			// -----------------------------------------
			// Fetch latest Laravel project policies
			// -----------------------------------------

			updatedPolicies, err := fetchProjectPolicies()
			if err != nil {
				log.Printf(
					"Policy synchronization failed. Skipping vulnerability scanning: %v",
					err,
				)

				// Fail closed: never scan using stale policies.
				continue
			}

			policies = updatedPolicies

			// -----------------------------------------
			// Filter projects using lifecycle + policy
			// -----------------------------------------

			scannableInventory := filterPolicyEligibleProjects(
				updatedInventory,
				policies,
				false,
			)

			// Remove projects that are stopped, missing,
			// archived, or no longer allowed to be scanned.
			reconcileKnownProjects(
				scannableInventory,
				knownProjects,
			)

			// -----------------------------------------
			// Determine scan trigger
			// -----------------------------------------

			now := time.Now().In(weeklyScanLocation)

			weeklyDue := weeklyScanDue(
				now,
				lastWeeklyScanKey,
			)

			newProjects := findNewProjects(
				scannableInventory,
				knownProjects,
			)

			var scanTargets []ProjectInventory
			scanReason := ""

			switch {
			case weeklyDue:
				// Weekly scan only includes projects
				// with weekly scanning enabled.
				scanTargets = filterPolicyEligibleProjects(
					updatedInventory,
					policies,
					true,
				)

				scanReason = "weekly scheduled scan"

			case len(newProjects) > 0:
				// Newly discovered or reactivated projects
				// only need scan_source permission.
				scanTargets = newProjects
				scanReason = "new or reactivated project discovery"

			default:
				fmt.Println("No vulnerability scan required.")
				continue
			}

			// -----------------------------------------
			// No weekly-eligible projects
			// -----------------------------------------

			if len(scanTargets) == 0 {
				if weeklyDue {
					// Avoid checking the same completed
					// weekly window every 30 seconds.
					lastWeeklyScanKey = weeklyScanKey(now)

					fmt.Println(
						"No projects are eligible for weekly scanning.",
					)
				} else {
					fmt.Println(
						"No running projects available for scanning.",
					)
				}

				continue
			}

			// -----------------------------------------
			// Retry protection
			// -----------------------------------------

			if !nextScanAttemptAt.IsZero() &&
				time.Now().Before(nextScanAttemptAt) {

				log.Printf(
					"Scan retry scheduled at %s",
					nextScanAttemptAt.Format(time.RFC3339),
				)

				continue
			}

			fmt.Println("\n--------------------------------")
			fmt.Printf("Scan trigger: %s\n", scanReason)
			fmt.Printf("Projects selected: %d\n", len(scanTargets))

			weeklyKey := weeklyScanKey(now)

			// -----------------------------------------
			// Run dependency scan
			// -----------------------------------------

			scanCtx, cancelScan := context.WithTimeout(
				ctx,
				60*time.Minute,
			)
			triggerType := "discovery"

			if weeklyDue {
				triggerType = "scheduled"
			}

			submission, scanErr := scanInventory(
				scanCtx,
				scanTargets,
				triggerType,
			)

			cancelScan()

			if scanErr != nil {
				log.Printf("Dependency scan failed: %v", scanErr)

				nextScanAttemptAt = time.Now().Add(
					scanRetryDelay,
				)

				continue
			}

			if len(submission.Projects) == 0 {
				log.Println("No scan results generated.")

				nextScanAttemptAt = time.Now().Add(
					scanRetryDelay,
				)

				continue
			}

			// -----------------------------------------
			// Submit scan results to Laravel
			// -----------------------------------------

			if err := sendScanResults(submission); err != nil {
				log.Printf(
					"Scan result submission failed: %v",
					err,
				)

				nextScanAttemptAt = time.Now().Add(
					scanRetryDelay,
				)

				continue
			}

			// -----------------------------------------
			// Update scheduling state
			// -----------------------------------------

			successful := successfullyScannedProjects(
				scanTargets,
				submission,
			)

			rememberProjects(
				successful,
				knownProjects,
			)

			if len(successful) == len(scanTargets) {
				if weeklyDue {
					lastWeeklyScanKey = weeklyKey
				}

				nextScanAttemptAt = time.Time{}
			} else {
				nextScanAttemptAt = time.Now().Add(
					scanRetryDelay,
				)

				log.Printf(
					"Some projects failed scanning. Retry after %s.",
					scanRetryDelay,
				)
			}

			fmt.Printf(
				"\nScan processing completed: %d/%d projects successful.\n",
				len(successful),
				len(scanTargets),
			)
		}
	}
}
