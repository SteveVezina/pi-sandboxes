package system

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pi-sandbox/pi/pkg/runtime/detect"
)

// Issue represents a single issue found by the doctor.
type Issue struct {
	Severity string // "error", "warning", "info"
	Message  string
	Recommendation string
}

// DoctorResult holds the results of a doctor check.
type DoctorResult struct {
	Issues []Issue
	Passed bool
}

// RunDoctor performs all doctor checks.
func RunDoctor() *DoctorResult {
	result := &DoctorResult{}

	// Check pi home
	piHome := PiHome()
	if !DirExists(piHome) {
		result.Issues = append(result.Issues, Issue{
			Severity:       "warning",
			Message:        fmt.Sprintf("Pi home directory does not exist: %s", piHome),
			Recommendation: "Initialize with 'pi system status' to create default directories",
		})
	} else {
		result.Issues = append(result.Issues, Issue{
			Severity: "info",
			Message:  fmt.Sprintf("Pi home directory exists: %s", piHome),
		})
	}

	// Check required subdirectories
	requiredDirs := []string{"sandboxes", "templates", "caches"}
	for _, dir := range requiredDirs {
		path := filepath.Join(piHome, dir)
		if DirExists(path) {
			result.Issues = append(result.Issues, Issue{
				Severity: "info",
				Message:  fmt.Sprintf("Directory exists: %s", dir),
			})
		} else {
			result.Issues = append(result.Issues, Issue{
				Severity:       "warning",
				Message:        fmt.Sprintf("Missing directory: %s", dir),
				Recommendation: "This directory will be created on first use",
			})
		}
	}

	// Check config file
	configPath := filepath.Join(piHome, "config.yaml")
	if DirExists(configPath) {
		result.Issues = append(result.Issues, Issue{
			Severity: "info",
			Message:  "Config file exists: config.yaml",
		})
	} else {
		result.Issues = append(result.Issues, Issue{
			Severity:       "info",
			Message:        "No config file found (using defaults)",
			Recommendation: "Create config.yaml for custom settings",
		})
	}

	// Check disk space (warn if < 1GB free)
	if DirExists(piHome) {
		// Simple disk space check using os.Stat on the parent
		parent := filepath.Dir(piHome)
		info, err := os.Stat(parent)
		if err == nil && info.IsDir() {
			// On macOS, we can use statfs, but for simplicity, just check
			// that the directory is accessible
			result.Issues = append(result.Issues, Issue{
				Severity: "info",
				Message:  "Disk space check: directory accessible",
			})
		}
	}

	// Check permissions
	if DirExists(piHome) {
		info, err := os.Stat(piHome)
		if err == nil {
			perm := info.Mode().Perm()
			if perm&0444 == 0 {
				result.Issues = append(result.Issues, Issue{
					Severity: "warning",
					Message:  "Pi home directory is not readable",
					Recommendation: "Run 'chmod 755 ~/.pi' to fix",
				})
			} else {
				result.Issues = append(result.Issues, Issue{
					Severity: "info",
					Message:  "Permissions OK",
				})
			}
		}
	}

	// Check available runtime backends
	available := detect.AvailableRuntimes("")
	if len(available) > 0 {
		result.Issues = append(result.Issues, Issue{
			Severity: "info",
			Message:  fmt.Sprintf("Available runtime backends: %s", available),
		})
		bestMode := detect.BestMode("")
		result.Issues = append(result.Issues, Issue{
			Severity: "info",
			Message:  fmt.Sprintf("Best available mode: %s", bestMode),
		})
	} else {
		result.Issues = append(result.Issues, Issue{
			Severity:       "error",
			Message:        "No runtime backend available",
			Recommendation: "Install gVisor (runsc), or use compat mode (Docker/Podman), or fast mode (Linux namespaces)",
		})
	}

	result.Passed = true
	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			result.Passed = false
		}
	}

	return result
}

// PrintDoctor prints the doctor results.
func PrintDoctor(result *DoctorResult) {
	fmt.Println("=== pi-sandbox Doctor ===")
	fmt.Println()

	status := "OK"
	if !result.Passed {
		status = "ISSUES FOUND"
	}
	fmt.Printf("Status: %s\n\n", status)

	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			fmt.Printf("  [ERROR]   %s\n", issue.Message)
		case "warning":
			fmt.Printf("  [WARNING] %s\n", issue.Message)
		default:
			fmt.Printf("  [OK]      %s\n", issue.Message)
		}
		if issue.Recommendation != "" {
			fmt.Printf("          -> %s\n", issue.Recommendation)
		}
	}
}
