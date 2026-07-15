// Package compat provides OCI container runtime support (Docker/Podman/runc).
package compat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Start starts the container.
func (c *Container) Start() error {
	if c == nil {
		return fmt.Errorf("nil container")
	}
	if c.Spec == nil {
		return fmt.Errorf("nil spec")
	}

	// Container is already started in CreateContainer with "sleep infinity"
	// Verify it's running
	return c.verifyRunning()
}

// verifyRunning checks if the container is running.
func (c *Container) verifyRunning() error {
	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "inspect", "-f", "{{.State.Status}}", c.Spec.Name)
	case RuntimePodman:
		cmd = exec.Command("podman", "inspect", "-f", "{{.State.Status}}", c.Spec.Name)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}

	status := strings.TrimSpace(string(output))
	if status != "running" {
		return fmt.Errorf("container is %s, not running", status)
	}

	c.Ready = true
	return nil
}

// Stop stops the container gracefully.
func (c *Container) Stop() error {
	if c == nil {
		return fmt.Errorf("nil container")
	}
	if c.Spec == nil {
		return fmt.Errorf("nil spec")
	}

	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "stop", "-t", "5", c.Spec.Name)
	case RuntimePodman:
		cmd = exec.Command("podman", "stop", "-t", "5", c.Spec.Name)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container might already be stopped
		return nil
	}
	_ = output

	c.Ready = false
	return nil
}

// Destroy destroys the container and cleans up.
func (c *Container) Destroy() error {
	if c == nil {
		return fmt.Errorf("nil container")
	}
	if c.Spec == nil {
		return fmt.Errorf("nil spec")
	}

	// Stop if running
	c.Stop()

	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "rm", "-f", c.Spec.Name)
	case RuntimePodman:
		cmd = exec.Command("podman", "rm", "-f", c.Spec.Name)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container might already be removed
		return nil
	}
	_ = output

	c.Ready = false
	return nil
}

// State returns the container state.
func (c *Container) State() string {
	if c == nil {
		return "unknown"
	}
	if !c.Ready {
		return "stopped"
	}

	rt := Best()
	if rt == nil {
		return "unknown"
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "inspect", "-f", "{{.State.Status}}", c.Spec.Name)
	case RuntimePodman:
		cmd = exec.Command("podman", "inspect", "-f", "{{.State.Status}}", c.Spec.Name)
	default:
		return "unknown"
	}

	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	return strings.TrimSpace(string(output))
}

// ContainerStatus returns info about a container.
type ContainerStatus struct {
	ID        string
	Name      string
	State     string
	Image     string
	Running   bool
	CreatedAt string
}

// ListContainers returns all pi-sandbox containers.
func ListContainers() ([]ContainerStatus, error) {
	rt := Best()
	if rt == nil {
		return nil, fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "ps", "-a", "--filter", "name=pi-sandbox-", "--format", "{{.ID}}|{{.Names}}|{{.Status}}|{{.Image}}")
	case RuntimePodman:
		cmd = exec.Command("podman", "ps", "-a", "--filter", "name=pi-sandbox-", "--format", "{{.ID}}|{{.Names}}|{{.Status}}|{{.Image}}")
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var result []ContainerStatus
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		status := ContainerStatus{
			ID:      parts[0],
			Name:    parts[1],
			State:   parts[2],
			Image:   parts[3],
			Running: strings.Contains(parts[2], "Up"),
		}
		result = append(result, status)
	}

	return result, nil
}

// PruneStale removes containers that are in a stale state.
func PruneStale() (int, error) {
	rt := Best()
	if rt == nil {
		return 0, fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "container", "prune", "-f", "--filter", "label=pi-sandbox=true")
	case RuntimePodman:
		cmd = exec.Command("podman", "container", "prune", "-f", "--filter", "label=pi-sandbox=true")
	default:
		return 0, fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("prune: %w: %s", err, string(output))
	}
	_ = output

	return 1, nil
}

// EnsureRuntimeDir ensures the runtime directory exists.
func EnsureRuntimeDir() error {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	piDir := fmt.Sprintf("%s/.pi-box/runtime", home)
	return os.MkdirAll(piDir, 0755)
}

// ContainerExists checks if a container exists.
func ContainerExists(name string) (bool, error) {
	rt := Best()
	if rt == nil {
		return false, fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "inspect", "-f", "{{.Name}}", name)
	case RuntimePodman:
		cmd = exec.Command("podman", "inspect", "-f", "{{.Name}}", name)
	default:
		return false, fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(string(output)) == "/"+name, nil
}

// ContainerHealthCheck performs a health check on the container.
func (c *Container) HealthCheck() error {
	if c == nil || c.Spec == nil {
		return fmt.Errorf("nil container or spec")
	}

	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "exec", c.Spec.Name, "test", "-d", "/workspace")
	case RuntimePodman:
		cmd = exec.Command("podman", "exec", c.Spec.Name, "test", "-d", "/workspace")
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	// Set timeout
	ctx, cancel := execCommandWithTimeout(10 * time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("health check failed: %w: %s", err, string(output))
	}

	return nil
}

// execCommandWithTimeout creates a command with a timeout context.
func execCommandWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
