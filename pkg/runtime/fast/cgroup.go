//go:build linux
// +build linux

package fast

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// CgroupConfig holds cgroup v2 resource limits.
type CgroupConfig struct {
	CPUPeriod   int64 // CPU period in microseconds (default: 100000)
	CPUQuota    int64 // CPU quota in microseconds (default: unlimited)
	MemoryLimit int64 // Memory limit in bytes (0 = unlimited)
	MaxPIDs     int   // Max processes (default: 256)
	IOReadBPS   int64 // Read bandwidth limit (0 = unlimited)
	IOWriteBPS  int64 // Write bandwidth limit (0 = unlimited)
}

// DefaultCgroupConfig returns default cgroup limits.
func DefaultCgroupConfig() *CgroupConfig {
	return &CgroupConfig{
		CPUPeriod:   100000,
		MemoryLimit: 0, // unlimited by default
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
	if err := os.MkdirAll(m.path, 0755); err != nil {
		return fmt.Errorf("create cgroup dir: %w", err)
	}
	return nil
}

// SetCPU sets CPU limits.
func (m *CgroupManager) SetCPU(period, quota int64) error {
	if period > 0 {
		if err := os.WriteFile(filepath.Join(m.path, "cpu.max"),
			[]byte(fmt.Sprintf("%d %d", quota, period)), 0644); err != nil {
			return fmt.Errorf("set cpu.max: %w", err)
		}
	}
	return nil
}

// SetMemory sets memory limit.
func (m *CgroupManager) SetMemory(limit int64) error {
	if limit > 0 {
		if err := os.WriteFile(filepath.Join(m.path, "memory.max"),
			[]byte(strconv.FormatInt(limit, 10)), 0644); err != nil {
			return fmt.Errorf("set memory.max: %w", err)
		}
	}
	return nil
}

// SetPIDs sets max processes.
func (m *CgroupManager) SetPIDs(max int) error {
	if err := os.WriteFile(filepath.Join(m.path, "pids.max"),
		[]byte(strconv.Itoa(max)), 0644); err != nil {
		return fmt.Errorf("set pids.max: %w", err)
	}
	return nil
}

// SetIO sets I/O bandwidth limits.
func (m *CgroupManager) SetIO(readBPS, writeBPS int64) error {
	if readBPS > 0 {
		if err := os.WriteFile(filepath.Join(m.path, "io.max"),
			[]byte(fmt.Sprintf("default read bps=%d", readBPS)), 0644); err != nil {
			return fmt.Errorf("set io.max: %w", err)
		}
	}
	if writeBPS > 0 {
		if err := os.WriteFile(filepath.Join(m.path, "io.max"),
			[]byte(fmt.Sprintf("default write bps=%d", writeBPS)), 0644); err != nil {
			return fmt.Errorf("set io.max: %w", err)
		}
	}
	return nil
}

// AddProcess adds a PID to the cgroup.
func (m *CgroupManager) AddProcess(pid int) error {
	return os.WriteFile(filepath.Join(m.path, "cgroup.procs"),
		[]byte(strconv.Itoa(pid)), 0644)
}

// Destroy removes the cgroup hierarchy.
func (m *CgroupManager) Destroy() error {
	return os.RemoveAll(m.path)
}

// Path returns the cgroup path.
func (m *CgroupManager) Path() string {
	return m.path
}
