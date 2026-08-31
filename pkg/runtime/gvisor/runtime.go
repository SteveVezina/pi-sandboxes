//go:build linux
// +build linux

package gvisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/oci"
)

const (
	// RuntimeName is the identifier for the gVisor runtime.
	RuntimeName = "gvisor"

	// DefaultImage is the default gVisor base image.
	DefaultImage = "gcr.io/gvisor/runsc:latest"

	// DefaultTimeout is the default command execution timeout.
	DefaultTimeout = 30 * time.Second
)

// Runtime implements the gVisor (runsc) backend using the shared OCI engine.
type Runtime struct {
	image   string
	timeout time.Duration
	eng     oci.Engine
	rootDir string
}

// New creates a gVisor runtime backed by the shared OCI engine.
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

// Mode returns the public mode this runtime serves.
func (r *Runtime) Mode() pruntime.Mode {
	return pruntime.ModeSecure
}

// IsAvailable checks if gVisor (runsc) is installed and usable.
func (r *Runtime) IsAvailable() bool {
	_, err := exec.LookPath("runsc")
	return err == nil
}

// Probe returns a capability report for this runtime.
func (r *Runtime) Probe(ctx context.Context) pruntime.CapabilityReport {
	report := pruntime.CapabilityReport{
		Available:        r.IsAvailable(),
		KernelBoundary:   true,
		Rootless:         false,
		UserNamespace:    true,
		Seccomp:          true,
		NetworkNamespace: true,
		EgressPolicy:     true,
		Snapshot:         true,
		WarmExec:         true,
		OCIImages:        true,
		HardwareVirt:     false,
		IsolationTier:    3,
		CompatTier:       3,
	}
	if !r.IsAvailable() {
		report.Reason = "gVisor (runsc) not installed"
		report.Missing = []string{"runsc"}
	}
	return report
}

// EnsureImage ensures the gVisor base image is available.
func (r *Runtime) EnsureImage(ctx context.Context) (string, error) {
	if r.eng == nil {
		return "", fmt.Errorf("OCI engine not initialized")
	}
	return r.eng.EnsureImage(ctx, oci.ImageRef(r.image))
}

// Create provisions a new sandbox using gVisor via the shared OCI engine.
func (r *Runtime) Create(ctx context.Context, spec pruntime.SandboxSpec) (pruntime.Handle, error) {
	if !r.IsAvailable() {
		return pruntime.Handle{}, fmt.Errorf("gVisor not available: %w", exec.ErrNotFound)
	}

	// Initialize OCI engine if not already done
	if r.eng == nil {
		r.eng = oci.NewEngine(oci.EngineConfig{
			Runtime: "runsc",
			RootDir: r.rootDir,
		})
	}

	// Ensure the gVisor base image is available
	imageID, err := r.eng.EnsureImage(ctx, oci.ImageRef(r.image))
	if err != nil {
		return pruntime.Handle{}, fmt.Errorf("ensure gVisor image: %w", err)
	}

	// Create the container using the shared OCI engine
	containerID, err := r.eng.Create(ctx, oci.ContainerSpec{
		ImageID:   imageID,
		SandboxID: spec.SandboxID,
		Workspace: spec.Workspace,
		Artifacts: spec.Artifacts,
		Caches:    spec.Caches,
		UserNS:    true,
		MountNS:   true,
		PIDNS:     true,
		Network:   spec.Network,
		Limits:    spec.Limits,
	})
	if err != nil {
		return pruntime.Handle{}, fmt.Errorf("create container: %w", err)
	}

	// Return handle with stable sandbox ID and runtime object ID
	return pruntime.Handle{
		SandboxID:       spec.SandboxID,
		RuntimeObjectID: containerID,
	}, nil
}

// Start starts a gVisor sandbox.
func (r *Runtime) Start(ctx context.Context, h pruntime.Handle) error {
	if r.eng == nil {
		return fmt.Errorf("OCI engine not initialized")
	}
	return r.eng.Start(ctx, h.RuntimeObjectID)
}

// Exec executes a command in a gVisor sandbox.
func (r *Runtime) Exec(ctx context.Context, h pruntime.Handle, req pruntime.ExecRequest) (pruntime.ExecSession, error) {
	if r.eng == nil {
		return nil, fmt.Errorf("OCI engine not initialized")
	}
	return r.eng.Exec(ctx, h.RuntimeObjectID, req)
}

// Inspect returns the state of a gVisor sandbox.
func (r *Runtime) Inspect(ctx context.Context, h pruntime.Handle) (pruntime.RuntimeState, error) {
	if r.eng == nil {
		return pruntime.RuntimeState{}, fmt.Errorf("OCI engine not initialized")
	}
	state, err := r.eng.Inspect(ctx, h.RuntimeObjectID)
	if err != nil {
		return pruntime.RuntimeState{}, fmt.Errorf("inspect container: %w", err)
	}
	return pruntime.RuntimeState{
		State:       state.State,
		ExitCode:    state.ExitCode,
		ContainerID: h.RuntimeObjectID,
	}, nil
}

// Stop stops a gVisor sandbox.
func (r *Runtime) Stop(ctx context.Context, h pruntime.Handle, grace time.Duration) error {
	if r.eng == nil {
		return fmt.Errorf("OCI engine not initialized")
	}
	return r.eng.Stop(ctx, h.RuntimeObjectID, grace)
}

// Destroy destroys a gVisor sandbox.
func (r *Runtime) Destroy(ctx context.Context, h pruntime.Handle) error {
	if r.eng == nil {
		return fmt.Errorf("OCI engine not initialized")
	}
	return r.eng.Remove(ctx, h.RuntimeObjectID)
}

// Stats returns resource statistics for a gVisor sandbox.
func (r *Runtime) Stats(ctx context.Context, h pruntime.Handle) (pruntime.RuntimeStats, error) {
	if r.eng == nil {
		return pruntime.RuntimeStats{}, fmt.Errorf("OCI engine not initialized")
	}
	stats, err := r.eng.Stats(ctx, h.RuntimeObjectID)
	if err != nil {
		return pruntime.RuntimeStats{}, fmt.Errorf("stats: %w", err)
	}
	return pruntime.RuntimeStats{
		MemoryUsageBytes:  stats.MemoryUsageBytes,
		CPUUsageNanoCores: stats.CPUUsageNanoCores,
	}, nil
}

// generateConfig generates a minimal runsc config.json (deprecated, kept for reference).
// This function is no longer used — gVisor now uses the shared OCI engine.
func (r *Runtime) generateConfig(id, bundleDir, template string) string {
	// Deprecated: this function is kept for reference only.
	// gVisor now uses the shared OCI engine for container creation.
	return `{
		"ociVersion": "1.0.2",
		"process": {
			"terminal": false,
			"user": {"uid": 1000, "gid": 1000},
			"args": ["/bin/sh", "-c", "sleep infinity"],
			"env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"]
		},
		"root": {"path": "rootfs", "readonly": false},
		"hostname": "` + id + `"
	}`
}

// ensureRuntimeDir ensures the runtime directory exists.
func ensureRuntimeDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, ".pi-box", "runtime")
	return os.MkdirAll(dir, 0755)
}
