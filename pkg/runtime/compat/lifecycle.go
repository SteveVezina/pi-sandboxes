package compat

import (
	"fmt"
	"os"
)

// Start starts the container.
func (c *Container) Start() error {
	if c == nil {
		return fmt.Errorf("nil container")
	}
	// In production: rt.Path to create/start container
	return nil
}

// Stop stops the container gracefully.
func (c *Container) Stop() error {
	if c == nil {
		return fmt.Errorf("nil container")
	}
	// In production: rt.Path to stop container
	return nil
}

// Destroy destroys the container and cleans up.
func (c *Container) Destroy() error {
	if c == nil {
		return fmt.Errorf("nil container")
	}

	// Stop if running
	c.Stop()

	// In production: rt.Path to remove container
	// For stub, just return nil
	return nil
}

// State returns the container state.
func (c *Container) State() string {
	if c == nil {
		return "unknown"
	}
	return "running"
}

// ContainerStatus returns info about a container.
type ContainerStatus struct {
	ID        string
	State     string
	Image     string
	Running   bool
	CreatedAt string
}

// ListContainers returns all containers.
func ListContainers() ([]ContainerStatus, error) {
	// In production: rt.Path to list containers
	// For stub, return empty list
	return []ContainerStatus{}, nil
}

// PruneStale removes containers that are in a stale state.
func PruneStale() (int, error) {
	// In production: find and remove stopped/broken containers
	return 0, nil
}

// EnsureRuntimeDir ensures the runtime directory exists.
func EnsureRuntimeDir() error {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	piDir := fmt.Sprintf("%s/.pi/runtime", home)
	return os.MkdirAll(piDir, 0755)
}
