package runtime

import (
	"context"
	"io"
	"time"
)

// Mode is a user-facing runtime mode (SPEC.md §13).
type Mode string

const (
	ModeFast     Mode = "fast"
	ModeCompat   Mode = "compat"
	ModeSecure   Mode = "secure"
	ModeIsolated Mode = "isolated"
	ModeMicroVM  Mode = "microvm"
	ModeAuto     Mode = "auto"
)

// Handle identifies a sandbox at the driver boundary. SandboxID is the
// stable user-facing identity; RuntimeObjectID is owned by the driver
// (container ID, VM ID, ...). SandboxID is never mutated after creation.
type Handle struct {
	SandboxID       string `json:"sandbox_id"`
	RuntimeObjectID string `json:"runtime_object_id"`
	DriverName      string `json:"driver_name"`
	Mode            Mode   `json:"mode"`
}

// ResourceLimits is the shared resource-control model applied by every
// driver at creation (SPEC.md §14.7.5).
type ResourceLimits struct {
	MemoryBytes     int64   `json:"memory_bytes,omitempty"`
	MemorySwapBytes int64   `json:"memory_swap_bytes,omitempty"`
	CPUs            float64 `json:"cpus,omitempty"`
	PIDs            int     `json:"pids,omitempty"`
	OpenFiles       int     `json:"open_files,omitempty"`
}

// SandboxSpec is the driver-facing creation request. Policy evaluation,
// template resolution, and path validation happen above this layer.
type SandboxSpec struct {
	SandboxID     string            `json:"sandbox_id"`
	Name          string            `json:"name"`
	Image         string            `json:"image,omitempty"`
	WorkspacePath string            `json:"workspace_path"`
	ArtifactsPath string            `json:"artifacts_path"`
	CachePaths    map[string]string `json:"cache_paths,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	NetworkMode   string            `json:"network_mode,omitempty"`
	Limits        ResourceLimits    `json:"limits"`
}

// ExecRequest describes one command execution inside a sandbox.
type ExecRequest struct {
	Command    []string          `json:"command"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	TTY        bool              `json:"tty,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
}

// ExecResult is the completion record of one execution.
type ExecResult struct {
	ExitCode  int           `json:"exit_code"`
	Duration  time.Duration `json:"duration"`
	TimedOut  bool          `json:"timed_out"`
	Truncated bool          `json:"truncated"`
}

// ExecSession is a running execution with streamed output.
type ExecSession interface {
	Stdout() io.Reader
	Stderr() io.Reader
	Wait(ctx context.Context) (ExecResult, error)
	Close() error
}

// RuntimeState is the driver-observed sandbox state, used for daemon
// startup reconciliation.
type RuntimeState struct {
	Status   string `json:"status"` // running | stopped | absent
	ExitCode int    `json:"exit_code,omitempty"`
}

// RuntimeStats reports live resource usage for one sandbox.
type RuntimeStats struct {
	MemoryBytes int64         `json:"memory_bytes"`
	CPUTime     time.Duration `json:"cpu_time"`
	PIDs        int           `json:"pids"`
}

// Driver is the lifecycle contract every isolation backend implements
// (SPEC.md §14.7.5, ADR-005). Drivers own only isolation, process creation,
// mounts, network attachment, resource controls, and termination.
type Driver interface {
	Name() string
	Mode() Mode
	Probe(ctx context.Context) CapabilityReport

	Create(ctx context.Context, spec SandboxSpec) (Handle, error)
	Start(ctx context.Context, h Handle) error
	Exec(ctx context.Context, h Handle, req ExecRequest) (ExecSession, error)
	Inspect(ctx context.Context, h Handle) (RuntimeState, error)
	Stop(ctx context.Context, h Handle, grace time.Duration) error
	Destroy(ctx context.Context, h Handle) error
	Stats(ctx context.Context, h Handle) (RuntimeStats, error)
}
