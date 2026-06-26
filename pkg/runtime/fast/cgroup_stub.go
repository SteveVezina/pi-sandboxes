//go:build !linux
// +build !linux

package fast

import (
	"fmt"
	"os"
	"path/filepath"
)

// CgroupConfig holds cgroup v2 resource limits.
type CgroupConfig struct {
	CPUPeriod   int64
	CPUQuota    int64
	MemoryLimit int64
	MaxPIDs     int
	IOReadBPS   int64
	IOWriteBPS  int64
}

// DefaultCgroupConfig returns default cgroup limits.
func DefaultCgroupConfig() *CgroupConfig {
	return &CgroupConfig{
		CPUPeriod:   100000,
		MemoryLimit: 0,
		MaxPIDs:     256,
	}
}

// CgroupManager manages a cgroup v2 hierarchy.
type CgroupManager struct {
	basePath string
	id       string
	path     string
}

// NewCgroupManager creates a new cgroup manager for the given sandbox ID.
func NewCgroupManager(basePath, id string) *CgroupManager {
	return &CgroupManager{
		basePath: basePath,
		id:       id,
		path:     filepath.Join(basePath, id),
	}
}

// Create creates the cgroup hierarchy.
func (m *CgroupManager) Create() error {
	return fmt.Errorf("cgroup v2 requires Linux")
}

// SetCPU sets CPU limits.
func (m *CgroupManager) SetCPU(period, quota int64) error {
	return fmt.Errorf("cgroup v2 requires Linux")
}

// SetMemory sets memory limit.
func (m *CgroupManager) SetMemory(limit int64) error {
	return fmt.Errorf("cgroup v2 requires Linux")
}

// SetPIDs sets max processes.
func (m *CgroupManager) SetPIDs(max int) error {
	return fmt.Errorf("cgroup v2 requires Linux")
}

// SetIO sets I/O bandwidth limits.
func (m *CgroupManager) SetIO(readBPS, writeBPS int64) error {
	return fmt.Errorf("cgroup v2 requires Linux")
}

// AddProcess adds a PID to the cgroup.
func (m *CgroupManager) AddProcess(pid int) error {
	return fmt.Errorf("cgroup v2 requires Linux")
}

// Destroy removes the cgroup hierarchy.
func (m *CgroupManager) Destroy() error {
	os.RemoveAll(m.path)
	return nil
}

// Path returns the cgroup path.
func (m *CgroupManager) Path() string {
	return m.path
}
