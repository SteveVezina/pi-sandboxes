package oci

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
)

// CLIEngine drives an OCI runtime through its CLI. Docker and Podman
// share argument shapes for every operation used here; runtime-specific
// differences (user mapping, seccomp delivery) live in securityArgs.
type CLIEngine struct {
	// Binary is the resolved CLI path.
	Binary string
	// runtimeName is "docker" or "podman".
	runtimeName string
	// Timeout bounds every CLI call.
	Timeout time.Duration
}

// NewDockerEngine returns a Docker-backed engine.
func NewDockerEngine(binary string) *CLIEngine {
	return &CLIEngine{Binary: binary, runtimeName: "docker", Timeout: DefaultCommandTimeout}
}

// NewPodmanEngine returns a Podman-backed engine.
func NewPodmanEngine(binary string) *CLIEngine {
	return &CLIEngine{Binary: binary, runtimeName: "podman", Timeout: DefaultCommandTimeout}
}

func (e *CLIEngine) Runtime() string { return e.runtimeName }

// mountOptions returns bind-mount options for workspace-class mounts.
// Exec stays allowed — coding agents run ./gradlew, node_modules/.bin/*,
// .venv/bin/python from these paths; noexec applies to /tmp and secret
// mounts only (SPEC.md §14.7.5). Docker Desktop on macOS/Windows doesn't
// support extra mount options.
func mountOptions() string {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return "rw"
	}
	return "rw,nosuid,nodev"
}

// securityArgs returns runtime-specific user mapping and the versioned
// seccomp profile flags.
func (e *CLIEngine) securityArgs() ([]string, error) {
	var args []string
	switch e.runtimeName {
	case "podman":
		// Rootless Podman: keep the invoking user's identity so bind
		// mounts stay writable without chown.
		uid, gid := os.Getuid(), os.Getgid()
		if uid == 0 {
			uid, gid = 1000, 1000
		}
		args = append(args, "--userns=keep-id", "--user", fmt.Sprintf("%d:%d", uid, gid))
		// Podman remote (macOS/Windows machine) resolves seccomp paths on
		// the server; only pass the host-written profile on native Linux.
		if runtime.GOOS != "linux" {
			return args, nil
		}
	default:
		// Docker reads the profile client-side and sends its content, so
		// the path works on every platform. Fixed unprivileged user per
		// the documented Docker mapping strategy (PROP-008).
		args = append(args, "--user", "1000:1000")
	}

	profile, err := SeccompProfilePath()
	if err != nil {
		return nil, err
	}
	return append(args, "--security-opt", "seccomp="+profile), nil
}

// limitArgs maps the shared resource-limit model onto CLI flags.
func limitArgs(l pruntime.ResourceLimits) []string {
	var args []string
	if l.MemoryBytes > 0 {
		args = append(args, "--memory", fmt.Sprintf("%d", l.MemoryBytes))
	}
	if l.MemorySwapBytes > 0 {
		args = append(args, "--memory-swap", fmt.Sprintf("%d", l.MemorySwapBytes))
	}
	if l.CPUs > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%g", l.CPUs))
	}
	if l.PIDs > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", l.PIDs))
	}
	if l.OpenFiles > 0 {
		args = append(args, "--ulimit", fmt.Sprintf("nofile=%d:%d", l.OpenFiles, l.OpenFiles))
	}
	return args
}

// createArgs builds the container creation arguments — the single place
// hardened defaults are encoded.
func (e *CLIEngine) createArgs(spec *ContainerSpec) ([]string, error) {
	networkMode := spec.NetworkMode
	if networkMode == "" {
		networkMode = "bridge"
	}
	args := []string{
		"run", "-d",
		"--name", spec.Name,
		"--label", "pi-sandbox=true",
		"--network", networkMode,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--read-only",
		"--tmpfs", "/tmp:rw,nosuid,noexec",
		"--tmpfs", "/home/agent:rw,nosuid",
	}

	security, err := e.securityArgs()
	if err != nil {
		return nil, err
	}
	args = append(args, security...)
	args = append(args, limitArgs(spec.Limits)...)

	opts := mountOptions()
	if spec.Workspace != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/workspace:%s", spec.Workspace, opts))
	}
	if spec.Artifacts != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/artifacts:%s", spec.Artifacts, opts))
	}
	for name, hostPath := range spec.Caches {
		args = append(args, "-v", fmt.Sprintf("%s:/cache/%s:%s", hostPath, name, opts))
	}

	return append(args, spec.Image, "/bin/sh", "-c", "sleep infinity"), nil
}

func (e *CLIEngine) run(ctx context.Context, args ...string) (string, error) {
	timeout := e.Timeout
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.Binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%s %s timed out: %w: %s", e.runtimeName, args[0], ctx.Err(), string(output))
		}
		return "", fmt.Errorf("%s %s: %w: %s", e.runtimeName, args[0], err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func (e *CLIEngine) Create(ctx context.Context, spec *ContainerSpec) (string, error) {
	if spec.Name == "" {
		return "", fmt.Errorf("container name is required")
	}
	if spec.Image == "" {
		return "", fmt.Errorf("container image is required")
	}
	args, err := e.createArgs(spec)
	if err != nil {
		return "", err
	}
	output, err := e.run(ctx, args...)
	if err != nil {
		return "", err
	}
	containerID := output
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}
	return containerID, nil
}

func (e *CLIEngine) Exec(ctx context.Context, name, command string) (*ExecResult, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, e.Binary, "exec", "-i", name, "/bin/sh", "-c", command)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
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
		DurationMs: time.Since(start).Milliseconds(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		TimedOut:   timedOut,
	}, nil
}

func (e *CLIEngine) ExecCmd(ctx context.Context, name, command string) Cmd {
	return exec.CommandContext(ctx, e.Binary, "exec", "-i", name, "/bin/sh", "-c", command)
}

func (e *CLIEngine) Inspect(ctx context.Context, name string) (string, error) {
	return e.run(ctx, "inspect", "-f", "{{.State.Status}}", name)
}

func (e *CLIEngine) Stop(ctx context.Context, name string, grace time.Duration) error {
	seconds := int(grace / time.Second)
	if seconds <= 0 {
		seconds = 5
	}
	_, err := e.run(ctx, "stop", "-t", fmt.Sprintf("%d", seconds), name)
	return err
}

func (e *CLIEngine) Remove(ctx context.Context, name string) error {
	_, err := e.run(ctx, "rm", "-f", name)
	return err
}

func (e *CLIEngine) Exists(ctx context.Context, name string) (bool, error) {
	output, err := e.run(ctx, "inspect", "-f", "{{.Name}}", name)
	if err != nil {
		return false, nil
	}
	return output == "/"+name || output == name, nil
}

func (e *CLIEngine) List(ctx context.Context) ([]ContainerStatus, error) {
	output, err := e.run(ctx, "ps", "-a",
		"--filter", "name=pi-sandbox-",
		"--format", "{{.ID}}|{{.Names}}|{{.Status}}|{{.Image}}")
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var result []ContainerStatus
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		result = append(result, ContainerStatus{
			ID:      parts[0],
			Name:    parts[1],
			State:   parts[2],
			Image:   parts[3],
			Running: strings.Contains(parts[2], "Up"),
		})
	}
	return result, nil
}

func (e *CLIEngine) Prune(ctx context.Context) error {
	_, err := e.run(ctx, "container", "prune", "-f", "--filter", "label=pi-sandbox=true")
	if err != nil {
		return fmt.Errorf("prune: %w", err)
	}
	return nil
}

func (e *CLIEngine) CopyFrom(ctx context.Context, name, src, dst string) error {
	_, err := e.run(ctx, "cp", name+":"+src, dst)
	if err != nil {
		return fmt.Errorf("copy from: %w", err)
	}
	return nil
}

func (e *CLIEngine) CopyTo(ctx context.Context, name, src, dst string) error {
	_, err := e.run(ctx, "cp", src, name+":"+dst)
	if err != nil {
		return fmt.Errorf("copy to: %w", err)
	}
	return nil
}

func (e *CLIEngine) Logs(ctx context.Context, name string, follow bool) (io.ReadCloser, error) {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, name)

	cmd := exec.CommandContext(ctx, e.Binary, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return stdout, nil
}
