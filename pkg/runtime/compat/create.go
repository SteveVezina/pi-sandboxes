// Package compat provides OCI container runtime support (Docker/Podman/runc).
package compat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var containerCommandTimeout = 2 * time.Minute

// ContainerSpec represents a sandbox container configuration.
type ContainerSpec struct {
	ID          string
	Name        string
	Image       string
	Workspace   string
	Artifacts   string
	Caches      map[string]string
	NetworkMode string
	Privileged  bool
	Template    string
	Mode        string
}

// Container represents a running OCI container.
type Container struct {
	ID    string
	Spec  *ContainerSpec
	Ready bool
	Host  string // host workspace dir (for bind mounts)
}

// CreateContainer creates a hardened OCI container from the spec.
func CreateContainer(spec *ContainerSpec) (*Container, error) {
	rt := Best()
	if rt == nil {
		return nil, fmt.Errorf("no OCI runtime found (try installing Docker, Podman, runc)")
	}

	// Validate spec
	if spec.ID == "" {
		return nil, fmt.Errorf("container ID is required")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("container image is required")
	}
	if spec.Name == "" {
		spec.Name = "pi-sandbox-" + spec.ID[:min(8, len(spec.ID))]
	}

	// Hardened defaults
	if spec.NetworkMode == "" {
		spec.NetworkMode = "bridge"
	}
	spec.Privileged = false

	// Validate mounts — never mount docker socket
	if spec.Workspace == "/var/run/docker.sock" {
		return nil, fmt.Errorf("cannot mount docker socket as workspace")
	}

	// Create the container using Docker/Podman CLI
	if err := createContainerWithCLI(rt, spec); err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	return &Container{
		ID:    spec.ID,
		Spec:  spec,
		Ready: true,
		Host:  spec.Workspace,
	}, nil
}

// createContainerWithCLI creates a container using the Docker/Podman CLI.
func createContainerWithCLI(rt *DetectedRuntime, spec *ContainerSpec) error {
	switch rt.Name {
	case RuntimeDocker:
		return createDockerContainer(spec)
	case RuntimePodman:
		return createPodmanContainer(spec)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}
}

// createDockerContainer creates a container using Docker CLI.
func createDockerContainer(spec *ContainerSpec) error {
	// Platform-specific mount options
	// Docker Desktop on macOS/Windows doesn't support noexec
	mountOpts := "rw"
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		mountOpts = "rw,nosuid,noexec,nodev"
	}

	args := []string{
		"run", "-d",
		"--name", spec.Name,
		"--rm",
		"--label", "pi-sandbox=true",
		"--network", spec.NetworkMode,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--read-only",
		"--tmpfs", "/tmp:rw,nosuid,noexec",
		"--tmpfs", "/home/agent:rw,nosuid,noexec",
	}

	// Add workspace mount
	if spec.Workspace != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/workspace:%s", spec.Workspace, mountOpts))
	}

	// Add artifacts mount
	if spec.Artifacts != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/artifacts:%s", spec.Artifacts, mountOpts))
	}

	// Add cache mounts
	for name, hostPath := range spec.Caches {
		args = append(args, "-v", fmt.Sprintf("%s:/cache/%s:%s", hostPath, name, mountOpts))
	}

	// Add image and command
	args = append(args, spec.Image, "/bin/sh", "-c", "sleep infinity")

	ctx, cancel := context.WithTimeout(context.Background(), containerCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("docker run timed out: %w: %s", ctx.Err(), string(output))
		}
		return fmt.Errorf("docker run: %w: %s", err, string(output))
	}

	// Extract container ID from output
	containerID := strings.TrimSpace(string(output))
	spec.ID = containerID[:min(12, len(containerID))]
	return nil
}

// createPodmanContainer creates a container using Podman CLI.
func createPodmanContainer(spec *ContainerSpec) error {
	// Platform-specific mount options
	mountOpts := "rw"
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		mountOpts = "rw,nosuid,noexec,nodev"
	}

	args := []string{
		"run", "-d",
		"--name", spec.Name,
		"--rm",
		"--label", "pi-sandbox=true",
		"--network", spec.NetworkMode,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--read-only",
		"--tmpfs", "/tmp:rw,nosuid,noexec",
		"--tmpfs", "/home/agent:rw,nosuid,noexec",
	}

	// Add workspace mount
	if spec.Workspace != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/workspace:%s", spec.Workspace, mountOpts))
	}

	// Add artifacts mount
	if spec.Artifacts != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/artifacts:%s", spec.Artifacts, mountOpts))
	}

	// Add cache mounts
	for name, hostPath := range spec.Caches {
		args = append(args, "-v", fmt.Sprintf("%s:/cache/%s:%s", hostPath, name, mountOpts))
	}

	// Add image and command
	args = append(args, spec.Image, "/bin/sh", "-c", "sleep infinity")

	ctx, cancel := context.WithTimeout(context.Background(), containerCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("podman run timed out: %w: %s", ctx.Err(), string(output))
		}
		return fmt.Errorf("podman run: %w: %s", err, string(output))
	}

	// Extract container ID from output
	containerID := strings.TrimSpace(string(output))
	spec.ID = containerID[:min(12, len(containerID))]
	return nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ContainerConfig generates the OCI container config from spec.
func ContainerConfig(spec *ContainerSpec) map[string]interface{} {
	config := map[string]interface{}{
		"ociVersion": "1.0.2",
		"process": map[string]interface{}{
			"terminal": false,
			"user":     map[string]interface{}{"uid": 1000, "gid": 1000},
			"args":     []string{"/bin/sh", "-c", "exec"},
			"env":      []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			"cwd":      "/workspace",
			"capabilities": map[string]interface{}{
				"bounding":    []string{},
				"effective":   []string{},
				"inheritable": []string{},
				"permitted":   []string{},
			},
			"noNewPrivileges": true,
		},
		"root": map[string]interface{}{
			"path":     "rootfs",
			"readonly": false,
		},
		"hostname": spec.ID,
		"linux": map[string]interface{}{
			"namespaces": []map[string]string{
				{"type": "pid"},
				{"type": "network"},
				{"type": "ipc"},
				{"type": "uts"},
				{"type": "mount"},
			},
			"resources": map[string]interface{}{
				"devices": []map[string]interface{}{
					{"allow": false},
				},
			},
			"maskedPaths": []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
			},
			"readonlyPaths": []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},
	}

	// Add mounts
	var mounts []map[string]interface{}
	if spec.Workspace != "" {
		mounts = append(mounts, map[string]interface{}{
			"type":        "bind",
			"source":      spec.Workspace,
			"destination": "/workspace",
			"options":     []string{"rw", "nosuid", "noexec", "nodev"},
		})
	}
	if spec.Artifacts != "" {
		mounts = append(mounts, map[string]interface{}{
			"type":        "bind",
			"source":      spec.Artifacts,
			"destination": "/artifacts",
			"options":     []string{"rw", "nosuid", "noexec", "nodev"},
		})
	}
	for name, hostPath := range spec.Caches {
		mounts = append(mounts, map[string]interface{}{
			"type":        "bind",
			"source":      hostPath,
			"destination": "/cache/" + name,
			"options":     []string{"rw", "nosuid", "noexec", "nodev"},
		})
	}
	config["mounts"] = mounts

	return config
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// JoinPaths joins paths safely.
func JoinPaths(paths ...string) string {
	return filepath.Join(paths...)
}
