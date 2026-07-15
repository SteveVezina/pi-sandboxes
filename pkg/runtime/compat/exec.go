// Package compat provides OCI container runtime support (Docker/Podman/runc).
package compat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/pi-sandbox/pi/pkg/runtime/oci"
)

// ExecResult holds the result of a container exec.
type ExecResult struct {
	ExitCode   int
	DurationMs int64
	Stdout     string
	Stderr     string
	TimedOut   bool
	Truncated  bool
}

// Exec executes a command in the container with streaming output.
func (c *Container) Exec(ctx context.Context, command string) (*ExecResult, error) {
	if c == nil {
		return nil, fmt.Errorf("nil container")
	}
	if c.Spec == nil {
		return nil, fmt.Errorf("nil spec")
	}

	eng, err := Engine()
	if err != nil {
		return nil, err
	}

	result, err := eng.Exec(ctx, c.Spec.Name, command)
	if err != nil {
		return nil, err
	}
	return &ExecResult{
		ExitCode:   result.ExitCode,
		DurationMs: result.DurationMs,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		TimedOut:   result.TimedOut,
	}, nil
}

// ExecStream executes a command in the container and streams output via a channel.
func (c *Container) ExecStream(ctx context.Context, command string, stdoutChan chan<- string, stderrChan chan<- string) error {
	if c == nil {
		return fmt.Errorf("nil container")
	}
	if c.Spec == nil {
		return fmt.Errorf("nil spec")
	}

	eng, err := Engine()
	if err != nil {
		return err
	}

	cmd := eng.ExecCmd(ctx, c.Spec.Name, command)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stdoutChan <- scanner.Text():
			}
		}
		close(stdoutChan)
	}()

	// Read stderr
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stderrChan <- scanner.Text():
			}
		}
		close(stderrChan)
	}()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("exit code %d", exitErr.ExitCode())
		}
		return err
	}
	return nil
}

// ExecCommand creates an exec command for the container.
func ExecCommand(rt *DetectedRuntime, containerID, command string) *exec.Cmd {
	if rt == nil {
		return exec.Command("/bin/sh", "-c", command)
	}

	switch rt.Name {
	case RuntimeDocker:
		return exec.Command("docker", "exec", "-i", containerID, "/bin/sh", "-c", command)
	case RuntimePodman:
		return exec.Command("podman", "exec", "-i", containerID, "/bin/sh", "-c", command)
	case RuntimeContainerd:
		return exec.Command("ctr", "-n", "tasks", "exec", "--exec-id", containerID, containerID, command)
	case RuntimeRunc:
		return exec.Command("runc", "exec", "-i", containerID, command)
	default:
		return exec.Command("/bin/sh", "-c", command)
	}
}

// ExecWithTimeout executes a command with a timeout.
func (c *Container) ExecWithTimeout(timeout time.Duration, command string) (*ExecResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Exec(ctx, command)
}

// CopyFrom copies files from the container to the host.
func (c *Container) CopyFrom(src, dst string) error {
	if c == nil || c.Spec == nil {
		return fmt.Errorf("nil container or spec")
	}
	eng, err := Engine()
	if err != nil {
		return err
	}
	return eng.CopyFrom(context.Background(), c.Spec.Name, src, dst)
}

// CopyTo copies files from the host to the container.
func (c *Container) CopyTo(src, dst string) error {
	if c == nil || c.Spec == nil {
		return fmt.Errorf("nil container or spec")
	}
	eng, err := Engine()
	if err != nil {
		return err
	}
	return eng.CopyTo(context.Background(), c.Spec.Name, src, dst)
}

// Logs returns the container logs.
func (c *Container) Logs(follow bool) (io.ReadCloser, error) {
	if c == nil || c.Spec == nil {
		return nil, fmt.Errorf("nil container or spec")
	}
	eng, err := Engine()
	if err != nil {
		return nil, err
	}
	return eng.Logs(context.Background(), c.Spec.Name, follow)
}

var _ oci.Engine = (*oci.CLIEngine)(nil)
