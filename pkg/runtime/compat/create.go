package compat

import (
	"fmt"
	"os"
	"path/filepath"
)

// ContainerSpec represents a container configuration.
type ContainerSpec struct {
	ID          string
	Image       string
	Workspace   string
	Artifacts   string
	Caches      map[string]string
	NetworkMode string
	Privileged  bool
}

// Container represents a running container.
type Container struct {
	ID   string
	Spec *ContainerSpec
}

// CreateContainer creates a hardened container from the spec.
func CreateContainer(spec *ContainerSpec) (*Container, error) {
	rt := Best()
	if rt == nil {
		return nil, fmt.Errorf("no OCI runtime found (try installing containerd, podman, runc, or docker)")
	}

	// Validate spec
	if spec.ID == "" {
		return nil, fmt.Errorf("container ID is required")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("container image is required")
	}

	// Hardened defaults
	if spec.NetworkMode == "" {
		spec.NetworkMode = "bridge"
	}
	spec.Privileged = false // Never privileged by default

	// Validate mounts — never mount docker socket
	if spec.Workspace == "/var/run/docker.sock" {
		return nil, fmt.Errorf("cannot mount docker socket as workspace")
	}

	// Build the container (stub — actual OCI container creation
	// would use the runtime's API or CLI)
	c := &Container{
		ID:   spec.ID,
		Spec: spec,
	}

	// In production, this would:
	// 1. Generate OCI bundle from spec
	// 2. Call rt.Path to create/start the container
	// 3. Return the container handle

	return c, nil
}

// ContainerConfig generates the OCI container config from spec.
func ContainerConfig(spec *ContainerSpec) map[string]interface{} {
	config := map[string]interface{}{
		"ociVersion": "1.0.2",
		"process": map[string]interface{}{
			"terminal":    false,
			"user":        map[string]interface{}{"uid": 1000, "gid": 1000},
			"args":        []string{"/bin/sh", "-c", "exec"},
			"env":         []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			"cwd":         "/workspace",
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
			"type":     "bind",
			"source":   spec.Workspace,
			"destination": "/workspace",
			"options":  []string{"rw", "nosuid", "noexec", "nodev"},
		})
	}
	if spec.Artifacts != "" {
		mounts = append(mounts, map[string]interface{}{
			"type":     "bind",
			"source":   spec.Artifacts,
			"destination": "/artifacts",
			"options":  []string{"rw", "nosuid", "noexec", "nodev"},
		})
	}
	for name, hostPath := range spec.Caches {
		mounts = append(mounts, map[string]interface{}{
			"type":     "bind",
			"source":   hostPath,
			"destination": "/cache/" + name,
			"options":  []string{"rw", "nosuid", "noexec", "nodev"},
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
