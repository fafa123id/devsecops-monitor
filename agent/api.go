package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func sendInventory(inventory []ProjectInventory) error {
	apiURL := strings.TrimRight(os.Getenv("DEVSECOPS_API_URL"), "/")
	token := os.Getenv("DEVSECOPS_AGENT_TOKEN")

	if apiURL == "" {
		return fmt.Errorf("DEVSECOPS_API_URL is not configured")
	}

	if token == "" {
		return fmt.Errorf("DEVSECOPS_AGENT_TOKEN is not configured")
	}

	payload := struct {
		SnapshotComplete bool               `json:"snapshot_complete"`
		Projects         []ProjectInventory `json:"projects"`
	}{
		SnapshotComplete: true,
		Projects:         inventory,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode inventory: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		apiURL+"/api/agent/inventory",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send inventory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}

	fmt.Printf("Inventory synchronized successfully. HTTP %d\n", resp.StatusCode)

	return nil
}
func sendHeartbeat() error {
	apiURL := strings.TrimRight(os.Getenv("DEVSECOPS_API_URL"), "/")
	token := os.Getenv("DEVSECOPS_AGENT_TOKEN")

	if apiURL == "" {
		return fmt.Errorf("DEVSECOPS_API_URL is not configured")
	}

	if token == "" {
		return fmt.Errorf("DEVSECOPS_AGENT_TOKEN is not configured")
	}

	payload := struct {
		AgentVersion string `json:"agent_version"`
	}{
		AgentVersion: "0.1.0",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode heartbeat: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		apiURL+"/api/agent/heartbeat",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{Timeout: 15 * time.Second}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("heartbeat API returned HTTP %d", resp.StatusCode)
	}

	fmt.Printf("Heartbeat sent successfully. HTTP %d\n", resp.StatusCode)

	return nil
}

func sendScanResults(submission ScanSubmission) error {
	apiURL := strings.TrimRight(os.Getenv("DEVSECOPS_API_URL"), "/")
	token := os.Getenv("DEVSECOPS_AGENT_TOKEN")

	if apiURL == "" {
		return fmt.Errorf("DEVSECOPS_API_URL is not configured")
	}
	if token == "" {
		return fmt.Errorf("DEVSECOPS_AGENT_TOKEN is not configured")
	}

	if err := validateScanSubmission(submission); err != nil {
		return fmt.Errorf("invalid scan submission: %w", err)
	}

	fmt.Println("\nPreparing scan submission...")
	fmt.Printf("Projects: %d\n", len(submission.Projects))

	totalEcosystems := 0
	totalFindings := 0

	for _, project := range submission.Projects {
		totalEcosystems += len(project.Ecosystems)

		for _, ecosystem := range project.Ecosystems {
			totalFindings += len(ecosystem.Findings)
		}
	}

	fmt.Printf("Ecosystems: %d\n", totalEcosystems)
	fmt.Printf("Dependency findings: %d\n", totalFindings)

	fmt.Println("Encoding scan results to JSON...")

	body, err := json.Marshal(submission)
	if err != nil {
		return fmt.Errorf("encode scan submission: %w", err)
	}

	fmt.Printf("Payload size: %.2f MB\n", float64(len(body))/1024/1024)
	fmt.Println("Sending scan results to Laravel API...")

	req, err := http.NewRequest(
		http.MethodPost,
		apiURL+"/api/agent/scans",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("create scan request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send scan results: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"scan API returned HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	fmt.Printf("Scan results submitted successfully. HTTP %d\n", resp.StatusCode)
	fmt.Printf("API response: %s\n", strings.TrimSpace(string(responseBody)))

	return nil
}
