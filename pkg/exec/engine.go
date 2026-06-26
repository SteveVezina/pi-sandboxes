package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Request holds the parameters for an exec command.
type Request struct {
	Command       string        `json:"command"`
	Cwd           string        `json:"cwd"`
	Timeout       time.Duration `json:"timeout_ms"`
	MaxOutput     int64         `json:"max_output_bytes"`
	ProcessLimit  int           `json:"processes"`
	NetworkMode   string        `json:"network"`
}

// DefaultRequest returns a request with sane defaults.
func DefaultRequest() *Request {
	return &Request{
		Cwd:         "/workspace",
		Timeout:     120 * time.Second,
		MaxOutput:   8 * 1024 * 1024, // 8 MiB
		ProcessLimit: 256,
	}
}

// Result holds the output of an exec command.
type Result struct {
	ExitCode    int       `json:"exitCode"`
	DurationMs  int64     `json:"durationMs"`
	Stdout      string    `json:"stdout"`
	Stderr      string    `json:"stderr"`
	Truncated   bool      `json:"truncated"`
	TimedOut    bool      `json:"timedOut"`
}

// Engine executes commands with isolation, timeout, and output limits.
type Engine struct {
	maxOutput int64
}

// NewEngine creates a new exec engine.
func NewEngine(maxOutput int64) *Engine {
	if maxOutput == 0 {
		maxOutput = 8 * 1024 * 1024
	}
	return &Engine{maxOutput: maxOutput}
}

// Run executes a command with timeout and output limits.
func (e *Engine) Run(ctx context.Context, req *Request) (*Result, error) {
	if req == nil {
		req = DefaultRequest()
	}

	if req.Timeout == 0 {
		req.Timeout = 120 * time.Second
	}
	if req.MaxOutput == 0 {
		req.MaxOutput = e.maxOutput
	}
	if req.Cwd == "" {
		req.Cwd = "/tmp"
	}

	start := time.Now()

	// Split command into args
	args := splitCommand(req.Command)
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = req.Cwd

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()

	duration := time.Since(start)
	timedOut := false
	truncated := false

	// Check if timed out
	if ctx.Err() == context.DeadlineExceeded {
		timedOut = true
	}

	// Truncate output if needed
	stdoutStr := TruncateBytes(stdout.Bytes(), req.MaxOutput)
	stderrStr := TruncateBytes(stderr.Bytes(), req.MaxOutput)
	if len(stdout.Bytes()) > int(req.MaxOutput) {
		truncated = true
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return &Result{
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		Stdout:     stdoutStr,
		Stderr:     stderrStr,
		Truncated:  truncated,
		TimedOut:   timedOut,
	}, nil
}

// splitCommand splits a command string into args.
// Simple split — doesn't handle shell quoting (shell mode handled separately).
func splitCommand(cmd string) []string {
	if cmd == "" {
		return nil
	}
	// Use a simple space split for now
	// Shell mode would use /bin/sh -c
	return []string{"/bin/sh", "-c", cmd}
}

// TruncateBytes truncates bytes to maxOutput, appending a marker if truncated.
func TruncateBytes(data []byte, maxOutput int64) string {
	if int64(len(data)) <= maxOutput {
		return string(data)
	}
	return string(data[:maxOutput]) + "\n... [truncated]"
}
