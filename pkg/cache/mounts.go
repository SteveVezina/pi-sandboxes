package cache

import (
	"fmt"
	"os"
	"path/filepath"
)

// MountPoint represents a cache mount inside the sandbox.
type MountPoint struct {
	HostPath    string `json:"hostPath"`
	SandboxPath string `json:"sandboxPath"`
	ReadOnly    bool   `json:"readOnly"`
}

// Manager manages cache mounts for sandbox sessions.
type Manager struct {
	scope Scope
}

// NewManager creates a cache manager for a scope.
func NewManager(scope Scope) *Manager {
	return &Manager{scope: scope}
}

// Mounts returns all cache mount points for this scope.
func (m *Manager) Mounts() ([]MountPoint, error) {
	var mounts []MountPoint

	for _, t := range AllCacheTypes() {
		hostPath := m.scope.Dir(t)
		sandboxPath := "/cache/" + string(t)

		// Ensure cache directory exists
		if err := os.MkdirAll(hostPath, 0755); err != nil {
			return nil, fmt.Errorf("ensure cache dir %s: %w", hostPath, err)
		}

		mounts = append(mounts, MountPoint{
			HostPath:    hostPath,
			SandboxPath: sandboxPath,
			ReadOnly:    false,
		})
	}

	return mounts, nil
}

// GetMount returns the mount point for a specific cache type.
func (m *Manager) GetMount(t Type) (*MountPoint, error) {
	hostPath := m.scope.Dir(t)
	if _, err := os.Stat(hostPath); err != nil {
		return nil, fmt.Errorf("cache %s not found: %w", t, err)
	}

	return &MountPoint{
		HostPath:    hostPath,
		SandboxPath: "/cache/" + string(t),
		ReadOnly:    false,
	}, nil
}

// Size returns the total size of all caches for this scope in bytes.
func (m *Manager) Size() (int64, error) {
	var total int64
	for _, t := range AllCacheTypes() {
		dir := m.scope.Dir(t)
		size, err := dirSize(dir)
		if err != nil {
			continue
		}
		total += size
	}
	return total, nil
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
