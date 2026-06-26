//go:build linux
// +build linux

package fast

import (
	"fmt"
	"os"
	"path/filepath"
)

// MountConfig defines the filesystem mounts for a sandbox.
type MountConfig struct {
	Rootfs       string   // Read-only root filesystem path
	Workspace    string   // Writable workspace path
	Artifacts    string   // Writable artifacts path
	Caches       map[string]string // cache name -> mount path
	Tmp          string   // /tmp path
	Home         string   // /home/agent path
	HostMounts   []HostMount // Explicit host mounts (opt-in)
}

// HostMount defines a host directory mount into the sandbox.
type HostMount struct {
	HostPath    string
	ContainerPath string
	ReadOnly    bool
}

// DefaultMountConfig returns the default mount configuration.
func DefaultMountConfig(rootDir, sandboxID string) *MountConfig {
	return &MountConfig{
		Rootfs:    filepath.Join(rootDir, "rootfs"),
		Workspace: filepath.Join(rootDir, "sandboxes", sandboxID, "workspace"),
		Artifacts: filepath.Join(rootDir, "sandboxes", sandboxID, "artifacts"),
		Caches: map[string]string{
			"npm":      filepath.Join(rootDir, "caches", "npm"),
			"pnpm":     filepath.Join(rootDir, "caches", "pnpm"),
			"pip":      filepath.Join(rootDir, "caches", "pip"),
			"uv":       filepath.Join(rootDir, "caches", "uv"),
			"go-mod":   filepath.Join(rootDir, "caches", "go-mod"),
			"go-build": filepath.Join(rootDir, "caches", "go-build"),
			"cargo":    filepath.Join(rootDir, "caches", "cargo"),
		},
		Tmp:    filepath.Join(rootDir, "sandboxes", sandboxID, "tmp"),
		Home:   filepath.Join(rootDir, "sandboxes", sandboxID, "home"),
	}
}

// EnsureDirectories creates all mount directories.
func (c *MountConfig) EnsureDirectories() error {
	dirs := []string{
		c.Rootfs, c.Workspace, c.Artifacts, c.Tmp, c.Home,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	for name, d := range c.Caches {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create cache dir %s (%s): %w", name, d, err)
		}
	}
	return nil
}

// Validate checks that no host directories are mounted by default.
func (c *MountConfig) Validate() error {
	if len(c.HostMounts) > 0 {
		return fmt.Errorf("host mounts are not allowed by default — user must opt in")
	}
	return nil
}

// WorkspaceDir returns the workspace path.
func (c *MountConfig) WorkspaceDir() string {
	return c.Workspace
}

// ArtifactsDir returns the artifacts path.
func (c *MountConfig) ArtifactsDir() string {
	return c.Artifacts
}
