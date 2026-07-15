// Package compat provides OCI container runtime support (Docker/Podman/runc).
package compat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
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

	start := time.Now()

	rt := Best()
	if rt == nil {
		return nil, fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.CommandContext(ctx, "docker", "exec", "-i", c.Spec.Name, "/bin/sh", "-c", command)
	case RuntimePodman:
		cmd = exec.CommandContext(ctx, "podman", "exec", "-i", c.Spec.Name, "/bin/sh", "-c", command)
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)
	timedOut := ctx.Err() == context.DeadlineExceeded

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if !timedOut {
			exitCode = 1
		}
	}

	return &ExecResult{
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		TimedOut:   timedOut,
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

	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.CommandContext(ctx, "docker", "exec", "-i", c.Spec.Name, "/bin/sh", "-c", command)
	case RuntimePodman:
		cmd = exec.CommandContext(ctx, "podman", "exec", "-i", c.Spec.Name, "/bin/sh", "-c", command)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	// Stream stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	// Stream stderr
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

	// Wait for command to complete
	err = cmd.Wait()
	if err != nil {
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

	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "cp", c.Spec.Name+":"+src, dst)
	case RuntimePodman:
		cmd = exec.Command("podman", "cp", c.Spec.Name+":"+src, dst)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy from: %w: %s", err, string(output))
	}

	return nil
}

// CopyTo copies files from the host to the container.
func (c *Container) CopyTo(src, dst string) error {
	if c == nil || c.Spec == nil {
		return fmt.Errorf("nil container or spec")
	}

	rt := Best()
	if rt == nil {
		return fmt.Errorf("no OCI runtime found")
	}

	var cmd *exec.Cmd
	switch rt.Name {
	case RuntimeDocker:
		cmd = exec.Command("docker", "cp", src, c.Spec.Name+":"+dst)
	case RuntimePodman:
		cmd = exec.Command("podman", "cp", src, c.Spec.Name+":"+dst)
	default:
		return fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy to: %w: %s", err, string(output))
	}

	return nil
}

// Logs returns the container logs.
func (c *Container) Logs(follow bool) (io.ReadCloser, error) {
	if c == nil || c.Spec == nil {
		return nil, fmt.Errorf("nil container or spec")
	}

	rt := Best()
	if rt == nil {
		return nil, fmt.Errorf("no OCI runtime found")
	}

	var args []string
	switch rt.Name {
	case RuntimeDocker:
		args = append(args, "logs")
		if follow {
			args = append(args, "-f")
		}
		args = append(args, c.Spec.Name)
	case RuntimePodman:
		args = append(args, "logs")
		if follow {
			args = append(args, "-f")
		}
		args = append(args, c.Spec.Name)
	default:
		return nil, fmt.Errorf("unsupported runtime: %s", rt.Name)
	}

	cmd := exec.Command(string(rt.Name), args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return stdout, nil
}
