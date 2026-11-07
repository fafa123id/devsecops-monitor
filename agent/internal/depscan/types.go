package depscan

import "strings"

// Scan statuses shared by all ecosystem scanners.
const (
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusUnsupported = "unsupported"
)

// Finding is the normalized vulnerability finding
// produced by every ecosystem scanner.
type Finding struct {
	FindingType      string         `json:"finding_type"`
	FindingID        string         `json:"finding_id"`
	VulnerabilityID  string         `json:"vulnerability_id,omitempty"`
	PackageName      string         `json:"package_name,omitempty"`
	InstalledVersion string         `json:"installed_version,omitempty"`
	FixedVersion     string         `json:"fixed_version,omitempty"`
	Severity         string         `json:"severity"`
	Title            string         `json:"title,omitempty"`
	Description      string         `json:"description,omitempty"`
	PrimaryURL       string         `json:"primary_url,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// EcosystemResult contains the result of scanning
// one dependency ecosystem.
type EcosystemResult struct {
	Ecosystem    string    `json:"ecosystem"`
	ManifestFile string    `json:"manifest_file"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Findings     []Finding `json:"findings"`
}

// Detection describes dependency manifests discovered
// inside a source directory.
type Detection struct {
	Present           bool
	Supported         bool
	ManifestFile      string
	LockFile          string
	UnsupportedReason string
}

// Source represents one unique project source directory.
type Source struct {
	Directory         string
	RelativeDirectory string
	Runtimes          []Runtime
}

// Runtime represents a container capable of accessing
// a particular source directory.
type Runtime struct {
	ContainerID   string
	ContainerName string
	ServiceName   string
	ContainerPath string
}

// CommandResult represents the result of executing
// an ecosystem command.
type CommandResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

func NormalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium", "moderate":
		return "medium"
	case "low":
		return "low"
	default:
		return "unknown"
	}
}
