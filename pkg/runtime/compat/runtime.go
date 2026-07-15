// Package compat provides OCI container runtime support (Docker/Podman/runc).
// It works across Linux, macOS (via Docker Desktop/Colima), and Windows (via WSL2/Docker).
package compat

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Runtime represents an OCI runtime.
type Runtime string

const (
	RuntimeDocker     Runtime = "docker"
	RuntimePodman     Runtime = "podman"
	RuntimeContainerd Runtime = "containerd"
	RuntimeRunc       Runtime = "runc"
)

// DetectedRuntime holds the detected runtime info.
type DetectedRuntime struct {
	Name    Runtime
	Path    string
	Version string
}

// DetectRuntime finds the first available OCI runtime and returns detailed info.
func DetectRuntime() *DetectedRuntime {
	runtimes := []Runtime{
		RuntimeContainerd,
		RuntimePodman,
		RuntimeRunc,
		RuntimeDocker,
	}

	for _, rt := range runtimes {
		if path, err := exec.LookPath(string(rt)); err == nil {
			version := detectRuntimeVersion(rt, path)
			return &DetectedRuntime{
				Name:    rt,
				Path:    path,
				Version: version,
			}
		}
	}

	return nil
}

// detectRuntimeVersion returns the version string for the given runtime.
func detectRuntimeVersion(rt Runtime, path string) string {
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	// Parse version from output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		// Return first line, truncated to 80 chars
		v := lines[0]
		if len(v) > 80 {
			v = v[:80] + "..."
		}
		return v
	}
	return "unknown"
}

// RuntimeName returns the string name of the runtime.
func (r Runtime) Name() string {
	return string(r)
}

// IsAvailable checks if a specific runtime is available.
func IsAvailable(rt Runtime) bool {
	_, err := exec.LookPath(string(rt))
	return err == nil
}

// Best returns the best available OCI runtime (cached).
var bestRuntime *DetectedRuntime
var bestRuntimeOnce sync.Once

// Best returns the best available OCI runtime.
func Best() *DetectedRuntime {
	bestRuntimeOnce.Do(func() {
		bestRuntime = DetectRuntime()
	})
	return bestRuntime
}

// RuntimeStatus holds runtime availability and version info.
type RuntimeStatus struct {
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}

// AllRuntimes returns status for all known OCI runtimes.
func AllRuntimes() []RuntimeStatus {
	names := []Runtime{
		RuntimeContainerd,
		RuntimePodman,
		RuntimeRunc,
		RuntimeDocker,
	}
	var result []RuntimeStatus
	for _, rt := range names {
		status := RuntimeStatus{Name: rt.Name()}
		if path, err := exec.LookPath(rt.Name()); err == nil {
			status.Available = true
			status.Path = path
			status.Version = detectRuntimeVersion(rt, path)
		}
		result = append(result, status)
	}
	return result
}

// EnsureRuntimeAvailable checks if at least one OCI runtime is available.
// Returns a helpful error message if none are found.
func EnsureRuntimeAvailable() error {
	if Best() != nil {
		return nil
	}
	return fmt.Errorf("no OCI runtime found — install Docker, Podman, runc, or containerd")
}
