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

// Manager manages cache mounts for sandboxes using daemon-managed volumes.
type Manager struct {
	scope   Scope
	rootDir string
}

// NewManager creates a cache manager for a scope.
func NewManager(scope Scope, rootDirs ...string) *Manager {
	rootDir := ""
	if len(rootDirs) > 0 {
		rootDir = rootDirs[0]
	}
	if rootDir == "" {
		rootDir = filepath.Join(os.Getenv("HOME"), ".pi-box")
	}
	return &Manager{scope: scope, rootDir: rootDir}
}

// Mounts returns all cache mount points for this scope using daemon-managed volumes.
func (m *Manager) Mounts() ([]MountPoint, error) {
	var mounts []MountPoint

	for _, t := range AllCacheTypes() {
		// Use daemon-managed cache storage instead of host bind mounts
		volumePath := m.volumePath(t)
		sandboxPath := "/cache/" + string(t)

		// Ensure volume directory exists
		if err := os.MkdirAll(volumePath, 0755); err != nil {
			return nil, fmt.Errorf("ensure volume dir %s: %w", volumePath, err)
		}

		mounts = append(mounts, MountPoint{
			HostPath:    volumePath,
			SandboxPath: sandboxPath,
			ReadOnly:    false,
		})
	}

	return mounts, nil
}

// volumePath returns the daemon-managed volume path for a cache type.
func (m *Manager) volumePath(cacheType Type) string {
	return filepath.Join(m.rootDir, "runtime", "caches", m.scope.String(), string(cacheType))
}

// GetMount returns the mount point for a specific cache type.
func (m *Manager) GetMount(t Type) (*MountPoint, error) {
	volumePath := m.volumePath(t)
	if _, err := os.Stat(volumePath); err != nil {
		return nil, fmt.Errorf("cache volume %s not found: %w", t, err)
	}

	return &MountPoint{
		HostPath:    volumePath,
		SandboxPath: "/cache/" + string(t),
		ReadOnly:    false,
	}, nil
}

// Size returns the total size of all cache volumes for this scope in bytes.
func (m *Manager) Size() (int64, error) {
	var total int64
	for _, t := range AllCacheTypes() {
		dir := m.volumePath(t)
		size, err := dirSize(dir)
		if err != nil {
			continue
		}
		total += size
	}
	return total, nil
}

// dirSize returns the total size of a directory in bytes.
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

// EnsureVolume ensures a cache volume exists and returns its path.
func (m *Manager) EnsureVolume(cacheType Type) (string, error) {
	volumePath := m.volumePath(cacheType)
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return "", fmt.Errorf("ensure volume dir %s: %w", volumePath, err)
	}
	return volumePath, nil
}

// VolumeExists checks if a cache volume exists.
func (m *Manager) VolumeExists(cacheType Type) bool {
	_, err := os.Stat(m.volumePath(cacheType))
	return err == nil
}

// RemoveVolume removes a cache volume.
func (m *Manager) RemoveVolume(cacheType Type) error {
	volumePath := m.volumePath(cacheType)
	return os.RemoveAll(volumePath)
}
