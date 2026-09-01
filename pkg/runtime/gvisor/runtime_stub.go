//go:build !linux
// +build !linux

package gvisor

import (
	"context"
	"fmt"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
)

const (
	// RuntimeName is the identifier for the gVisor runtime.
	RuntimeName = "gvisor"

	// DefaultImage is the default gVisor base image.
	DefaultImage = "gcr.io/gvisor/runsc:latest"

	// DefaultTimeout is the default command execution timeout.
	DefaultTimeout = 30 * time.Second
)

// Runtime is the non-Linux stub: gVisor/runsc only exists on Linux, so every
// lifecycle method fails with an actionable error (F18). The exported
// surface mirrors runtime.go (linux) exactly — same Driver contract on
// every platform, so callers never need a build-tag switch of their own.
type Runtime struct {
	image   string
	timeout time.Duration
	rootDir string
}

var _ pruntime.Driver = (*Runtime)(nil)

// New creates a new gVisor runtime.
func New(image, rootDir string, timeout time.Duration) *Runtime {
	if image == "" {
		image = DefaultImage
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &Runtime{image: image, timeout: timeout, rootDir: rootDir}
}

// Default creates a gVisor runtime with sensible defaults.
func Default(rootDir string) *Runtime {
	return New("", rootDir, 0)
}

// Name returns the runtime name.
func (r *Runtime) Name() string { return RuntimeName }

// Mode returns the public mode this runtime serves.
func (r *Runtime) Mode() pruntime.Mode { return pruntime.ModeSecure }

// IsAvailable always returns false on non-Linux.
func (r *Runtime) IsAvailable() bool { return false }

// Probe reports gVisor as unavailable on non-Linux.
func (r *Runtime) Probe(ctx context.Context) pruntime.CapabilityReport {
	return pruntime.CapabilityReport{
		Available: false,
		Reason:    "gVisor (runsc) requires Linux",
		Missing:   []string{"linux"},
	}
}

func errNotLinux() error {
	return fmt.Errorf("gVisor not available on non-Linux platforms")
}

func (r *Runtime) Create(ctx context.Context, spec pruntime.SandboxSpec) (pruntime.Handle, error) {
	return pruntime.Handle{}, errNotLinux()
}

func (r *Runtime) Start(ctx context.Context, h pruntime.Handle) error {
	return errNotLinux()
}

func (r *Runtime) Exec(ctx context.Context, h pruntime.Handle, req pruntime.ExecRequest) (pruntime.ExecSession, error) {
	return nil, errNotLinux()
}

func (r *Runtime) Inspect(ctx context.Context, h pruntime.Handle) (pruntime.RuntimeState, error) {
	return pruntime.RuntimeState{Status: "absent"}, errNotLinux()
}

func (r *Runtime) Stop(ctx context.Context, h pruntime.Handle, grace time.Duration) error {
	return errNotLinux()
}

func (r *Runtime) Destroy(ctx context.Context, h pruntime.Handle) error {
	return errNotLinux()
}

func (r *Runtime) Stats(ctx context.Context, h pruntime.Handle) (pruntime.RuntimeStats, error) {
	return pruntime.RuntimeStats{}, errNotLinux()
}
