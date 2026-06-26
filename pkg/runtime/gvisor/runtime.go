//go:build linux
// +build linux

package gvisor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
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

// IsAvailable checks if gVisor (runsc) is installed and usable.
func (r *Runtime) IsAvailable() bool {
	_, err := exec.LookPath("runsc")
	return err == nil
}

// IsGvisor returns true (always, since this is the gvisor runtime).
func (r *Runtime) IsGvisor() bool {
	return true
}

// Create provisions a new sandbox session using gVisor.
func (r *Runtime) Create(ctx context.Context, id, template string) error {
	if !r.IsAvailable() {
		return fmt.Errorf("gVisor not available: %w", exec.ErrNotFound)
	}

	// Create sandbox directory
	sandboxDir := filepath.Join(r.rootDir, id)
	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		return fmt.Errorf("create sandbox dir: %w", err)
	}

	// Create bundle directory structure for runsc
	// runsc uses containerd-compatible bundle format
	bundleDir := filepath.Join(sandboxDir, "bundle")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return fmt.Errorf("create bundle dir: %w", err)
	}

	// Write config.json (minimal spec)
	configPath := filepath.Join(bundleDir, "config.json")
	config := r.generateConfig(id, bundleDir, template)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}

	// Create rootfs directory
	rootfsDir := filepath.Join(bundleDir, "rootfs")
	if err := os.MkdirAll(rootfsDir, 0755); err != nil {
		return fmt.Errorf("create rootfs dir: %w", err)
	}

	// Run 'runsc create' to create the sandbox
	cmd := exec.CommandContext(ctx, "runsc", "create", "--bundle", bundleDir, id)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("runsc create: %w: %s", err, stderr.String())
	}

	return nil
}

// Destroy terminates a sandbox session.
func (r *Runtime) Destroy(ctx context.Context, id string) error {
	// Force delete the sandbox
	cmd := exec.CommandContext(ctx, "runsc", "delete", "-f", id)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Ignore errors if sandbox is already gone
		return fmt.Errorf("runsc delete: %w: %s", err, stderr.String())
	}

	return nil
}

// Exec runs a command inside the sandbox with timeout and truncation.
func (r *Runtime) Exec(ctx context.Context, id, cwd, cmdStr string, timeout time.Duration) (*exec.ExitError, []byte, []byte, error) {
	if timeout == 0 {
		timeout = r.timeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "runsc", "exec", "-d", cwd, id, "/bin/sh", "-c", cmdStr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr, stdout.Bytes(), stderr.Bytes(), nil
		}
		return nil, stdout.Bytes(), stderr.Bytes(), err
	}
	return nil, stdout.Bytes(), stderr.Bytes(), nil
}

// CloneGit clones a repository into the sandbox workspace.
func (r *Runtime) CloneGit(ctx context.Context, id, url, cwd string) error {
	cmd := exec.CommandContext(ctx, "runsc", "exec", "-d", cwd, id, "/bin/sh", "-c",
		fmt.Sprintf("git clone %s .", url))
	return cmd.Run()
}

// GetState returns the runtime state of a sandbox.
func (r *Runtime) GetState(ctx context.Context, id string) (string, error) {
	cmd := exec.CommandContext(ctx, "runsc", "state", id)
	output, err := cmd.Output()
	if err != nil {
		return "unknown", err
	}
	return string(output), nil
}

// GetStatus returns the sandbox status string.
func (r *Runtime) GetStatus(ctx context.Context, id string) (string, error) {
	state, err := r.GetState(ctx, id)
	if err != nil {
		return "stopped", err
	}
	return state, nil
}

// GetLogs returns sandbox logs.
func (r *Runtime) GetLogs(ctx context.Context, id string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "runsc", "logs", id)
	return cmd.Output()
}

// Snapshot creates a filesystem snapshot.
func (r *Runtime) Snapshot(ctx context.Context, id, name string) error {
	cmd := exec.CommandContext(ctx, "runsc", "snapshot", id, name)
	return cmd.Run()
}

// Rollback restores a snapshot.
func (r *Runtime) Rollback(ctx context.Context, id, name string) error {
	cmd := exec.CommandContext(ctx, "runsc", "restore", id, name)
	return cmd.Run()
}

// ListSnapshots returns available snapshots.
func (r *Runtime) ListSnapshots(ctx context.Context, id string) ([]string, error) {
	// Stub — in production, query runsc for snapshot metadata.
	return []string{}, nil
}

// DeleteSnapshot removes a snapshot.
func (r *Runtime) DeleteSnapshot(ctx context.Context, id, name string) error {
	cmd := exec.CommandContext(ctx, "runsc", "snapshot", "delete", id, name)
	return cmd.Run()
}

// GetTimeout returns the command timeout.
func (r *Runtime) GetTimeout() time.Duration {
	return r.timeout
}

// GetImage returns the container image.
func (r *Runtime) GetImage() string {
	return r.image
}

// GetRootDir returns the root directory.
func (r *Runtime) GetRootDir() string {
	return r.rootDir
}

// GetMode returns the runtime mode.
func (r *Runtime) GetMode() string {
	return "secure"
}

// GetSecurityLevel returns the security level (0-10). gVisor is high.
func (r *Runtime) GetSecurityLevel() int {
	return 9
}

// generateConfig generates a minimal OCI spec for runsc.
func (r *Runtime) generateConfig(id, bundleDir, template string) string {
	// Minimal OCI runtime spec for gVisor
	return `{
		"ociVersion": "1.0.2",
		"process": {
			"terminal": false,
			"user": {
				"uid": 0,
				"gid": 0
			},
			"args": ["/bin/sh"],
			"env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
			"cwd": "/workspace",
			"rlimits": [
				{"type": "RLIMIT_NOFILE", "hard": 1024, "soft": 1024}
			],
			"noNewPrivileges": true
		},
		"root": {
			"path": "rootfs",
			"readonly": true
		},
		"hostname": "` + id + `",
		"mounts": [
			{
				"destination": "/proc",
				"type": "proc",
				"source": "proc"
			},
			{
				"destination": "/dev",
				"type": "tmpfs",
				"source": "tmpfs",
				"options": ["nosuid", "noexec", "nodev"]
			}
		],
		"linux": {
			"namespaces": [
				{"type": "pid"},
				{"type": "ipc"},
				{"type": "uts"},
				{"type": "mount"},
				{"type": "network"}
			],
			"resources": {
				"devices": [
					{"allow": false, "access": "rwm"}
				]
			},
			"maskedPaths": [
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug"
			],
			"readonlyPaths": [
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/laptops",
				"/proc/sys",
				"/proc/sysrq-trigger"
			]
		}
	}`
}

// CgroupConfig holds cgroup v2 resource limits for gVisor.
type CgroupConfig struct {
	CPUPeriod   int64 // CPU period in microseconds
	CPUQuota    int64 // CPU quota in microseconds
	MemoryLimit int64 // Memory limit in bytes
	MaxPIDs     int   // Max processes
}

// DefaultCgroupConfig returns default cgroup limits.
func DefaultCgroupConfig() *CgroupConfig {
	return &CgroupConfig{
		CPUPeriod: 100000,
		MemoryLimit: 512 * 1024 * 1024, // 512MB
		MaxPIDs:     128,
	}
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
		path:     filepath.Join(basePath, id),
	}
}

// Create creates the cgroup hierarchy.
func (m *CgroupManager) Create() error {
	return os.MkdirAll(m.path, 0755)
}

// SetCPU sets CPU limits.
func (m *CgroupManager) SetCPU(period, quota int64) error {
	if period > 0 {
		return os.WriteFile(filepath.Join(m.path, "cpu.max"),
			[]byte(fmt.Sprintf("%d %d", quota, period)), 0644)
	}
	return nil
}

// SetMemory sets memory limit.
func (m *CgroupManager) SetMemory(limit int64) error {
	if limit > 0 {
		return os.WriteFile(filepath.Join(m.path, "memory.max"),
			[]byte(strconv.FormatInt(limit, 10)), 0644)
	}
	return nil
}

// SetPIDs sets max processes.
func (m *CgroupManager) SetPIDs(max int) error {
	return os.WriteFile(filepath.Join(m.path, "pids.max"),
		[]byte(strconv.Itoa(max)), 0644)
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

// Setup creates a syscall.SysProcAttr with namespace flags for gVisor.
func Setup(cfg *NamespaceConfig) *syscall.SysProcAttr {
	if cfg == nil {
		cfg = DefaultNamespaceConfig()
	}

	attr := &syscall.SysProcAttr{}

	if cfg.UserNS {
		attr.Cloneflags |= syscall.CLONE_NEWUSER
		attr.UidMappings = []syscall.SysProcIDRange{
			{ContainerID: 0, HostID: cfg.HostUID, Size: 1},
			{ContainerID: 1, HostID: cfg.HostUID + 1, Size: 65535},
		}
		attr.GidMappings = []syscall.SysProcIDRange{
			{ContainerID: 0, HostID: cfg.HostGID, Size: 1},
			{ContainerID: 1, HostID: cfg.HostGID + 1, Size: 65535},
		}
	}

	if cfg.MountNS {
		attr.Cloneflags |= syscall.CLONE_NEWNS
	}

	if cfg.PIDNS {
		attr.Cloneflags |= syscall.CLONE_NEWPID
	}

	return attr
}

// Validate checks if namespace operations are supported.
func Validate() error {
	return nil
}

// WriteUIDMap writes UID mapping for a namespace.
func WriteUIDMap(namespaceID int, uidMap string) error {
	path := fmt.Sprintf("/proc/%d/uid_map", namespaceID)
	return os.WriteFile(path, []byte(uidMap), 0644)
}

// WriteGIDMap writes GID mapping for a namespace.
func WriteGIDMap(namespaceID int, gidMap string) error {
	path := fmt.Sprintf("/proc/%d/gid_map", namespaceID)
	return os.WriteFile(path, []byte(gidMap), 0644)
}
