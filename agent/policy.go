package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ProjectPolicy struct {
	DockerProjectName string `json:"docker_project_name"`
	Status            string `json:"status"`
	ScanSource        bool   `json:"scan_source"`
	WeeklyScanEnabled bool   `json:"weekly_scan_enabled"`
}

type projectPolicyResponse struct {
	Policies []ProjectPolicy `json:"policies"`
}

func policyProjectKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// fetchProjectPolicies retrieves project scanning policies from Laravel.
// An error means the agent must not perform vulnerability scanning.
func fetchProjectPolicies() (map[string]ProjectPolicy, error) {
	apiURL := strings.TrimRight(
		os.Getenv("DEVSECOPS_API_URL"),
		"/",
	)

	token := os.Getenv("DEVSECOPS_AGENT_TOKEN")

	if apiURL == "" {
		return nil, fmt.Errorf("DEVSECOPS_API_URL is not configured")
	}

	if token == "" {
		return nil, fmt.Errorf("DEVSECOPS_AGENT_TOKEN is not configured")
	}

	req, err := http.NewRequest(
		http.MethodGet,
		apiURL+"/api/agent/policies",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create policy request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch project policies: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(
		io.LimitReader(resp.Body, 1024*1024),
	)
	if err != nil {
		return nil, fmt.Errorf("read policy response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"policy API returned HTTP %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	var response projectPolicyResponse

	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode project policies: %w", err)
	}

	if response.Policies == nil {
		return nil, fmt.Errorf(
			"policy API response does not contain a valid policies array",
		)
	}

	policies := make(map[string]ProjectPolicy)

	for _, policy := range response.Policies {
		key := policyProjectKey(policy.DockerProjectName)

		if key == "" {
			return nil, fmt.Errorf(
				"policy API returned an empty project name",
			)
		}

		if _, exists := policies[key]; exists {
			return nil, fmt.Errorf(
				"policy API returned duplicate project policy: %s",
				policy.DockerProjectName,
			)
		}

		policy.Status = strings.ToLower(
			strings.TrimSpace(policy.Status),
		)

		policies[key] = policy
	}

	fmt.Printf(
		"Project policies retrieved successfully: %d\n",
		len(policies),
	)

	return policies, nil
}
