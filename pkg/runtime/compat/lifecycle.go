// Package compat provides OCI container runtime support (Docker/Podman/runc).
package compat

import (
	"context"
	"fmt"
	"os"
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
	eng, err := Engine()
	if err != nil {
		return err
	}

	status, err := eng.Inspect(context.Background(), c.Spec.Name)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}
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

	eng, err := Engine()
	if err != nil {
		return err
	}

	if err := eng.Stop(context.Background(), c.Spec.Name, 5*time.Second); err != nil {
		return fmt.Errorf("stop container %s: %w", c.Spec.Name, err)
	}

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

	eng, err := Engine()
	if err != nil {
		return err
	}

	if err := eng.Remove(context.Background(), c.Spec.Name); err != nil {
		return fmt.Errorf("remove container %s: %w", c.Spec.Name, err)
	}

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

	eng, err := Engine()
	if err != nil {
		return "unknown"
	}

	status, err := eng.Inspect(context.Background(), c.Spec.Name)
	if err != nil {
		return "unknown"
	}
	return status
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
	eng, err := Engine()
	if err != nil {
		return nil, err
	}

	list, err := eng.List(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]ContainerStatus, 0, len(list))
	for _, item := range list {
		result = append(result, ContainerStatus{
			ID:      item.ID,
			Name:    item.Name,
			State:   item.State,
			Image:   item.Image,
			Running: item.Running,
		})
	}
	return result, nil
}

// PruneStale removes containers that are in a stale state.
func PruneStale() (int, error) {
	eng, err := Engine()
	if err != nil {
		return 0, err
	}
	if err := eng.Prune(context.Background()); err != nil {
		return 0, err
	}
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
	eng, err := Engine()
	if err != nil {
		return false, err
	}
	return eng.Exists(context.Background(), name)
}

// RemoveManagedVolumes removes all daemon-managed volumes for a sandbox ID.
func RemoveManagedVolumes(sandboxID string) error {
	eng, err := Engine()
	if err != nil {
		return err
	}
	return eng.RemoveVolumes(context.Background(), sandboxID)
}

// ContainerHealthCheck performs a health check on the container.
func (c *Container) HealthCheck() error {
	if c == nil || c.Spec == nil {
		return fmt.Errorf("nil container or spec")
	}

	eng, err := Engine()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := eng.Exec(ctx, c.Spec.Name, "test -d /workspace")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("health check failed: exit code %d: %s", result.ExitCode, result.Stderr)
	}
	return nil
}
