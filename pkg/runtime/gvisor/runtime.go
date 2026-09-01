//go:build linux
// +build linux

package gvisor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/runtime/oci"
)

const (
	// RuntimeName is the identifier for the gVisor runtime.
	RuntimeName = "gvisor"

	// DefaultImage is the default gVisor base image.
	DefaultImage = "gcr.io/gvisor/runsc:latest"

	// DefaultTimeout is the default command execution timeout.
	DefaultTimeout = 30 * time.Second

	// ociRuntime is the `--runtime` value that selects gVisor under Docker.
	ociRuntime = "runsc"
)

// Runtime implements the gVisor (runsc) backend using the shared OCI engine.
// gVisor runs as a Docker/Podman `--runtime=runsc` selection, not a
// standalone CLI, so container lifecycle goes through oci.CLIEngine exactly
// like the compat driver (ADR-005) — only the extra --runtime flag differs.
type Runtime struct {
	image   string
	timeout time.Duration
	eng     oci.Engine
	rootDir string
}

var _ pruntime.Driver = (*Runtime)(nil)

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

// IsAvailable checks if gVisor (runsc) and a Docker/Podman CLI able to
// drive it are both installed.
func (r *Runtime) IsAvailable() bool {
	if _, err := exec.LookPath("runsc"); err != nil {
		return false
	}
	_, err := exec.LookPath("docker")
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
		report.Reason = "gVisor (runsc) or docker not installed"
		report.Missing = []string{"runsc", "docker"}
	}
	return report
}

// engine lazily resolves the docker-runsc engine. Docker pulls a missing
// image implicitly on `run`, so there is no separate "ensure image" step
// (unlike the pre-ADR-005 implementation this replaces).
func (r *Runtime) engine() (oci.Engine, error) {
	if r.eng != nil {
		return r.eng, nil
	}
	path, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("gVisor requires docker on PATH: %w", err)
	}
	e := oci.NewDockerEngineWithRuntime(path, ociRuntime)
	e.Timeout = r.timeout
	r.eng = e
	return r.eng, nil
}

// Create provisions a new sandbox using gVisor via the shared OCI engine.
func (r *Runtime) Create(ctx context.Context, spec pruntime.SandboxSpec) (pruntime.Handle, error) {
	if !r.IsAvailable() {
		return pruntime.Handle{}, fmt.Errorf("gVisor not available: %w", exec.ErrNotFound)
	}

	eng, err := r.engine()
	if err != nil {
		return pruntime.Handle{}, err
	}

	image := spec.Image
	if image == "" {
		image = r.image
	}

	containerID, err := eng.Create(ctx, &oci.ContainerSpec{
		SandboxID: spec.SandboxID,
		Name:      compat.ContainerName(spec.SandboxID),
		Image:     image,
		Workspace: spec.WorkspacePath,
		Artifacts: spec.ArtifactsPath,
		Caches:    spec.CachePaths,
		Network:   spec.Network,
		Limits:    spec.Limits,
	})
	if err != nil {
		return pruntime.Handle{}, fmt.Errorf("create gVisor container: %w", err)
	}

	return pruntime.Handle{
		SandboxID:       spec.SandboxID,
		RuntimeObjectID: containerID,
		DriverName:      RuntimeName,
		Mode:            pruntime.ModeSecure,
	}, nil
}

// Start is a no-op: CLIEngine.Create already runs the container (`docker
// run -d`). It confirms the container came up.
func (r *Runtime) Start(ctx context.Context, h pruntime.Handle) error {
	eng, err := r.engine()
	if err != nil {
		return err
	}
	state, err := eng.Inspect(ctx, h.RuntimeObjectID)
	if err != nil {
		return fmt.Errorf("start: inspect gVisor container: %w", err)
	}
	if state != "running" {
		return fmt.Errorf("start: gVisor container %s is %q, not running", h.RuntimeObjectID, state)
	}
	return nil
}

// Exec executes a command in a gVisor sandbox.
func (r *Runtime) Exec(ctx context.Context, h pruntime.Handle, req pruntime.ExecRequest) (pruntime.ExecSession, error) {
	eng, err := r.engine()
	if err != nil {
		return nil, err
	}

	execCtx := ctx
	var cancel context.CancelFunc
	if req.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	result, err := eng.Exec(execCtx, h.RuntimeObjectID, shellCommand(req))
	if err != nil {
		return nil, fmt.Errorf("exec in gVisor container: %w", err)
	}
	return newCompletedSession(*result), nil
}

// shellCommand renders an ExecRequest as a single /bin/sh -c string
// (CLIEngine.Exec's surface), applying WorkingDir and Env inline since the
// engine only accepts one command string.
func shellCommand(req pruntime.ExecRequest) string {
	var b strings.Builder
	if req.WorkingDir != "" {
		b.WriteString("cd " + shellQuote(req.WorkingDir) + " && ")
	}
	for k, v := range req.Env {
		b.WriteString(k + "=" + shellQuote(v) + " ")
	}
	parts := make([]string, len(req.Command))
	for i, c := range req.Command {
		parts[i] = shellQuote(c)
	}
	b.WriteString(strings.Join(parts, " "))
	return b.String()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// completedSession adapts oci.ExecResult (blocking, already-finished) to
// the streaming pruntime.ExecSession contract.
type completedSession struct {
	stdout *bytes.Reader
	stderr *bytes.Reader
	result pruntime.ExecResult
}

func newCompletedSession(r oci.ExecResult) *completedSession {
	return &completedSession{
		stdout: bytes.NewReader([]byte(r.Stdout)),
		stderr: bytes.NewReader([]byte(r.Stderr)),
		result: pruntime.ExecResult{
			ExitCode: r.ExitCode,
			Duration: time.Duration(r.DurationMs) * time.Millisecond,
			TimedOut: r.TimedOut,
		},
	}
}

func (s *completedSession) Stdout() io.Reader { return s.stdout }
func (s *completedSession) Stderr() io.Reader { return s.stderr }
func (s *completedSession) Wait(ctx context.Context) (pruntime.ExecResult, error) {
	return s.result, nil
}
func (s *completedSession) Close() error { return nil }

// Inspect returns the state of a gVisor sandbox.
func (r *Runtime) Inspect(ctx context.Context, h pruntime.Handle) (pruntime.RuntimeState, error) {
	eng, err := r.engine()
	if err != nil {
		return pruntime.RuntimeState{}, err
	}
	state, err := eng.Inspect(ctx, h.RuntimeObjectID)
	if err != nil {
		return pruntime.RuntimeState{}, fmt.Errorf("inspect gVisor container: %w", err)
	}
	status := "stopped"
	switch state {
	case "running":
		status = "running"
	case "":
		status = "absent"
	}
	return pruntime.RuntimeState{Status: status}, nil
}

// Stop stops a gVisor sandbox.
func (r *Runtime) Stop(ctx context.Context, h pruntime.Handle, grace time.Duration) error {
	eng, err := r.engine()
	if err != nil {
		return err
	}
	return eng.Stop(ctx, h.RuntimeObjectID, grace)
}

// Destroy destroys a gVisor sandbox.
func (r *Runtime) Destroy(ctx context.Context, h pruntime.Handle) error {
	eng, err := r.engine()
	if err != nil {
		return err
	}
	return eng.Remove(ctx, h.RuntimeObjectID)
}

// Stats returns resource statistics for a gVisor sandbox. Not yet
// implemented — the shared oci.Engine interface has no stats call
// (F18 gap; tracked in docs/features/F18-secure-mode-gvisor.md).
func (r *Runtime) Stats(ctx context.Context, h pruntime.Handle) (pruntime.RuntimeStats, error) {
	return pruntime.RuntimeStats{}, fmt.Errorf("gvisor: Stats not implemented")
}
