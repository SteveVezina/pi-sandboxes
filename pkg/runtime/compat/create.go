// Package compat provides OCI container runtime support (Docker/Podman/runc).
package compat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pi-sandbox/pi/pkg/runtime/oci"
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
// ID is the stable session identity; RuntimeObjectID is the engine-owned
// container ID. The session ID is never mutated after creation.
type Container struct {
	ID              string
	RuntimeObjectID string
	Spec            *ContainerSpec
	Ready           bool
	Host            string // host workspace dir (for bind mounts)
}

// Engine returns the shared OCI engine for the detected runtime.
func Engine() (oci.Engine, error) {
	rt := Best()
	if rt == nil {
		return nil, fmt.Errorf("no OCI runtime found (try installing Docker, Podman, runc)")
	}
	return engineFor(rt)
}

func engineFor(rt *DetectedRuntime) (oci.Engine, error) {
	switch rt.Name {
	case RuntimeDocker:
		e := oci.NewDockerEngine(rt.Path)
		e.Timeout = containerCommandTimeout
		return e, nil
	case RuntimePodman:
		e := oci.NewPodmanEngine(rt.Path)
		e.Timeout = containerCommandTimeout
		return e, nil
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", rt.Name)
	}
}

// CreateContainer creates a hardened OCI container from the spec.
func CreateContainer(spec *ContainerSpec) (*Container, error) {
	eng, err := Engine()
	if err != nil {
		return nil, err
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

	containerID, err := eng.Create(context.Background(), &oci.ContainerSpec{
		Name:        spec.Name,
		Image:       spec.Image,
		Workspace:   spec.Workspace,
		Artifacts:   spec.Artifacts,
		Caches:      spec.Caches,
		NetworkMode: spec.NetworkMode,
	})
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	return &Container{
		ID:              spec.ID,
		RuntimeObjectID: containerID,
		Spec:            spec,
		Ready:           true,
		Host:            spec.Workspace,
	}, nil
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
