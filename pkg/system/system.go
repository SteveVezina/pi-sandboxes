package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PiHome returns the ~/.pi-box/ directory path.
func PiHome() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".pi-box")
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DirSize returns the total size of a directory in bytes.
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// FormatSize formats a byte size into human-readable form.
func FormatSize(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GiB", float64(bytes)/(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.1f MiB", float64(bytes)/(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.1f KiB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d B", bytes)
}

// DiskInfo holds disk space information.
type DiskInfo struct {
	Total int64
	Free  int64
	Used  int64
}

// GetDiskInfo returns disk space info for the given path.
func GetDiskInfo(path string) (*DiskInfo, error) {
	// Simple implementation — just check if path exists
	// Full disk info requires syscall on Unix
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &DiskInfo{
		Total: info.Size(),
		Free:  0, // Not implemented in MVP
		Used:  0, // Not implemented in MVP
	}, nil
}

// TimeAgo returns a human-readable string for time ago.
func TimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", d/time.Minute)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", d/time.Hour)
	}
	return fmt.Sprintf("%dd ago", d/(24*time.Hour))
}

// RuntimeBackend holds detailed info about a single runtime backend,
// mirroring the daemon's capability report (SPEC.md §14.7.5).
type RuntimeBackend struct {
	Name          string   `json:"mode"`
	Available     bool     `json:"available"`
	Reason        string   `json:"reason,omitempty"`
	Missing       []string `json:"missing,omitempty"`
	Description   string   `json:"description"`
	IsolationTier int      `json:"isolation_tier"`
	CompatTier    int      `json:"compat_tier"`
}

// RuntimeInfo holds runtime backend information from the daemon.
type RuntimeInfo struct {
	Available []RuntimeBackend `json:"available"`
	Best      string           `json:"best"`
}

// GetRuntimes fetches runtime backend information from the daemon.
func GetRuntimes(socketPath string) (*RuntimeInfo, error) {
	url := "http://localhost/v1/system/runtimes"
	if socketPath != "" {
		// Use curl via exec for unix socket access
		cmdStr := fmt.Sprintf("curl -s -H 'Content-Type: application/json' --unix-socket %s %s", socketPath, url)
		output, err := exec.CommandContext(context.Background(), "sh", "-c", cmdStr).Output()
		if err != nil {
			return nil, fmt.Errorf("runtimes: %w", err)
		}
		var result struct {
			Available []string         `json:"available"`
			Best      string           `json:"best"`
			Backends  []RuntimeBackend `json:"backends"`
		}
		if err := json.Unmarshal(output, &result); err != nil {
			return nil, fmt.Errorf("parse runtimes: %w", err)
		}
		return &RuntimeInfo{
			Available: result.Backends,
			Best:      result.Best,
		}, nil
	}
	// Fallback: HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("runtimes: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("runtimes: HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Available []string         `json:"available"`
		Best      string           `json:"best"`
		Backends  []RuntimeBackend `json:"backends"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse runtimes: %w", err)
	}
	return &RuntimeInfo{
		Available: result.Backends,
		Best:      result.Best,
	}, nil
}

// PrintRuntimes prints runtime backend information to stdout.
func PrintRuntimes(info *RuntimeInfo) {
	fmt.Println("Runtime backends:")
	fmt.Println("────────────────────────────────────────────────────────────────")
	for _, rt := range info.Available {
		status := "✗ unavailable"
		if rt.Available {
			status = "✓ available"
			if rt.Name == info.Best {
				status += " (best)"
			}
		}
		fmt.Printf("  %-12s %s  (isolation tier %d, compat tier %d)\n", rt.Name, status, rt.IsolationTier, rt.CompatTier)
		fmt.Printf("             %s\n", rt.Description)
		if !rt.Available && rt.Reason != "" {
			fmt.Printf("             reason: %s\n", rt.Reason)
			if len(rt.Missing) > 0 {
				fmt.Printf("             missing: %s\n", strings.Join(rt.Missing, ", "))
			}
		}
	}
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Printf("Best available mode: %s\n", info.Best)
}

// execCommand is a wrapper around exec.Command for testing.
var execCommand = exec.CommandContext
