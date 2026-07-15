//go:build !linux
// +build !linux

package gvisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	// RuntimeName is the identifier for the gVisor runtime.
	RuntimeName = "gvisor"

	// DefaultImage is the default gVisor base image.
	DefaultImage = "gcr.io/gvisor/runsc:latest"

	// DefaultTimeout is the default command execution timeout.
	DefaultTimeout = 30 * time.Second
)

// Runtime implements the gVisor (runsc) backend.
type Runtime struct {
	image   string
	timeout time.Duration
	rootDir string
}

// New creates a new gVisor runtime.
func New(image, rootDir string, timeout time.Duration) *Runtime {
	if image == "" {
		image = DefaultImage
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &Runtime{
		image:   image,
		timeout: timeout,
		rootDir: rootDir,
	}
}

// Default creates a gVisor runtime with sensible defaults.
func Default(rootDir string) *Runtime {
	return New("", rootDir, 0)
}

// Name returns the runtime name.
func (r *Runtime) Name() string {
	return RuntimeName
}

// IsAvailable always returns false on non-Linux.
func (r *Runtime) IsAvailable() bool {
	return false
}

// Create always returns an error on non-Linux.
func (r *Runtime) Create(ctx context.Context, id, template string) error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

// Destroy always returns an error on non-Linux.
func (r *Runtime) Destroy(ctx context.Context, id string) error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

// Exec always returns an error on non-Linux.
func (r *Runtime) Exec(ctx context.Context, id, cwd, cmdStr string, timeout time.Duration) (*exec.ExitError, []byte, []byte, error) {
	return nil, nil, nil, fmt.Errorf("gVisor not available on non-Linux platforms")
}

// CloneGit always returns an error on non-Linux.
func (r *Runtime) CloneGit(ctx context.Context, id, url, cwd string) error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

// GetState always returns an error on non-Linux.
func (r *Runtime) GetState(ctx context.Context, id string) (string, error) {
	return "unknown", fmt.Errorf("gVisor not available on non-Linux platforms")
}

// GetStatus always returns stopped on non-Linux.
func (r *Runtime) GetStatus(ctx context.Context, id string) (string, error) {
	return "stopped", fmt.Errorf("gVisor not available on non-Linux platforms")
}

// GetLogs always returns an error on non-Linux.
func (r *Runtime) GetLogs(ctx context.Context, id string) ([]byte, error) {
	return nil, fmt.Errorf("gVisor not available on non-Linux platforms")
}

// Snapshot always returns an error on non-Linux.
func (r *Runtime) Snapshot(ctx context.Context, id, name string) error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

// Rollback always returns an error on non-Linux.
func (r *Runtime) Rollback(ctx context.Context, id, name string) error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

// ListSnapshots returns empty on non-Linux.
func (r *Runtime) ListSnapshots(ctx context.Context, id string) ([]string, error) {
	return []string{}, nil
}

// DeleteSnapshot always returns an error on non-Linux.
func (r *Runtime) DeleteSnapshot(ctx context.Context, id, name string) error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

// GetTimeout returns the timeout.
func (r *Runtime) GetTimeout() time.Duration {
	return r.timeout
}

// GetImage returns the image.
func (r *Runtime) GetImage() string {
	return r.image
}

// GetRootDir returns the root dir.
func (r *Runtime) GetRootDir() string {
	return r.rootDir
}

// GetMode returns "secure".
func (r *Runtime) GetMode() string {
	return "secure"
}

// DefaultCgroupConfig returns default cgroup limits.
func DefaultCgroupConfig() *CgroupConfig {
	return &CgroupConfig{
		CPUPeriod:   100000,
		MemoryLimit: 512 * 1024 * 1024,
		MaxPIDs:     128,
	}
}

// CgroupConfig holds cgroup v2 resource limits for gVisor.
type CgroupConfig struct {
	CPUPeriod   int64
	CPUQuota    int64
	MemoryLimit int64
	MaxPIDs     int
}

// CgroupManager manages cgroup v2 for gVisor sandboxes.
type CgroupManager struct {
	basePath string
	id       string
	path     string
}

// NewCgroupManager creates a new cgroup manager.
func NewCgroupManager(basePath, id string) *CgroupManager {
	return &CgroupManager{
		basePath: basePath,
		id:       id,
		path:     basePath + "/" + id,
	}
}

// Create creates the cgroup hierarchy.
func (m *CgroupManager) Create() error {
	return os.MkdirAll(m.path, 0755)
}

// SetCPU sets CPU limits.
func (m *CgroupManager) SetCPU(period, quota int64) error {
	return nil
}

// SetMemory sets memory limit.
func (m *CgroupManager) SetMemory(limit int64) error {
	return nil
}

// SetPIDs sets max processes.
func (m *CgroupManager) SetPIDs(max int) error {
	return nil
}

// AddProcess adds a PID to the cgroup.
func (m *CgroupManager) AddProcess(pid int) error {
	return nil
}

// Destroy removes the cgroup hierarchy.
func (m *CgroupManager) Destroy() error {
	return os.RemoveAll(m.path)
}

// Path returns the cgroup path.
func (m *CgroupManager) Path() string {
	return m.path
}

// NamespaceConfig holds namespace configuration.
type NamespaceConfig struct {
	UserNS  bool
	MountNS bool
	PIDNS   bool
	HostUID int
	HostGID int
}

// DefaultNamespaceConfig returns default namespace config.
func DefaultNamespaceConfig() *NamespaceConfig {
	return &NamespaceConfig{
		UserNS:  true,
		MountNS: true,
		PIDNS:   true,
		HostUID: 1000,
		HostGID: 1000,
	}
}

// Setup creates a SysProcAttr with namespace flags.
// On non-Linux, returns a no-op attribute.
func Setup(cfg *NamespaceConfig) interface{} {
	if cfg == nil {
		cfg = DefaultNamespaceConfig()
	}
	return &NamespaceConfig{}
}

// Validate checks if namespace operations are supported.
func Validate() error {
	return nil
}

// WriteUIDMap writes UID mapping for a namespace.
func WriteUIDMap(namespaceID int, uidMap string) error {
	return nil
}

// WriteGIDMap writes GID mapping for a namespace.
func WriteGIDMap(namespaceID int, gidMap string) error {
	return nil
}
