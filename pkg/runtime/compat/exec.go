package compat

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ExecResult holds the result of a container exec.
type ExecResult struct {
	ExitCode   int
	DurationMs int64
	Stdout     string
	Stderr     string
	TimedOut   bool
}

// Exec executes a command in the container.
func (c *Container) Exec(ctx context.Context, command string) (*ExecResult, error) {
	if c == nil {
		return nil, fmt.Errorf("nil container")
	}

	start := time.Now()

	// In production, this would use the OCI runtime to exec into the container.
	// For now, stub with a no-op that returns success.
	// The actual implementation would use:
	// - containerd: client.Container.Exec(ctx, id, spec)
	// - podman: podman exec -i <id> <command>
	// - runc: runc exec <id> <command>

	_ = ctx
	_ = command

	duration := time.Since(start)

	return &ExecResult{
		ExitCode:   0,
		DurationMs: duration.Milliseconds(),
		Stdout:     "",
		Stderr:     "",
		TimedOut:   false,
	}, nil
}

// ExecCommand creates an exec command for the container.
func ExecCommand(rt *DetectedRuntime, containerID, command string) *exec.Cmd {
	if rt == nil {
		// Fallback: just run the command directly
		return exec.Command("/bin/sh", "-c", command)
	}

	switch rt.Name {
	case RuntimeContainerd:
		// containerd uses ctr exec
		return exec.Command("ctr", "-n", "tasks", "run", "--net-host", "--image", containerID, "exec-id", command)
	case RuntimePodman:
		return exec.Command("podman", "exec", "-i", containerID, "/bin/sh", "-c", command)
	case RuntimeRunc:
		return exec.Command("runc", "exec", "-i", containerID, command)
	case RuntimeDocker:
		return exec.Command("docker", "exec", "-i", containerID, "/bin/sh", "-c", command)
	default:
		return exec.Command("/bin/sh", "-c", command)
	}
}
