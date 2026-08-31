// Package oci provides the shared OCI engine used by the compat and
// secure runtime modes (SPEC.md §14.7.5, ADR-005). Docker/Podman argument
// construction lives here, in exactly one place per operation.
package oci

import (
	"context"
	"io"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
)

// DefaultCommandTimeout bounds runtime CLI calls so a stalled OCI runtime
// fails instead of wedging the daemon.
const DefaultCommandTimeout = 2 * time.Minute

// ContainerSpec is the engine-facing container creation request.
// The caller owns sandbox identity; the engine returns the runtime
// object ID and never mutates the caller's identifiers.
type ContainerSpec struct {
	// SandboxID is the stable user-facing sandbox identity. The engine
	// uses it as the egress-proxy auth identity (ADR-006); it never
	// becomes the runtime object ID.
	SandboxID string
	Name      string
	Image     string
	Workspace string
	Artifacts string
	Caches    map[string]string
	Network   pruntime.NetworkSpec
	Limits    pruntime.ResourceLimits
}

// ExecResult holds the result of a container exec.
type ExecResult struct {
	ExitCode   int
	DurationMs int64
	Stdout     string
	Stderr     string
	TimedOut   bool
}

// ContainerStatus describes one container in a listing.
type ContainerStatus struct {
	ID      string
	Name    string
	State   string
	Image   string
	Running bool
}

// Engine is the shared OCI lifecycle used by compat and secure modes.
type Engine interface {
	// Runtime returns the engine implementation name ("docker", "podman").
	Runtime() string
	// Create creates and starts a warm container, returning the runtime
	// container ID.
	Create(ctx context.Context, spec *ContainerSpec) (string, error)
	// Exec runs a shell command in the container and captures output.
	Exec(ctx context.Context, name, command string) (*ExecResult, error)
	// ExecCmd returns a prepared exec command for streaming callers.
	ExecCmd(ctx context.Context, name, command string) Cmd
	// Inspect returns the container status ("running", "exited", ...).
	Inspect(ctx context.Context, name string) (string, error)
	// Stop stops the container with a grace period.
	Stop(ctx context.Context, name string, grace time.Duration) error
	// Remove force-removes the container.
	Remove(ctx context.Context, name string) error
	// Exists reports whether the named container exists.
	Exists(ctx context.Context, name string) (bool, error)
	// List returns all pi-sandbox containers.
	List(ctx context.Context) ([]ContainerStatus, error)
	// Prune removes stopped pi-sandbox containers.
	Prune(ctx context.Context) error
	// RemoveVolumes removes daemon-managed volumes whose name contains
	// the given filter (e.g. a sandbox ID).
	RemoveVolumes(ctx context.Context, filter string) error
	// CopyFrom copies a path from the container to the host.
	CopyFrom(ctx context.Context, name, src, dst string) error
	// CopyTo copies a path from the host into the container.
	CopyTo(ctx context.Context, name, src, dst string) error
	// Logs returns a reader over container logs.
	Logs(ctx context.Context, name string, follow bool) (io.ReadCloser, error)
}

// Cmd is the minimal surface streaming callers need from a prepared
// command (satisfied by *exec.Cmd).
type Cmd interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}
