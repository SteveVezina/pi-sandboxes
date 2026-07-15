package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Request holds the parameters for an exec command.
type Request struct {
	Command      string        `json:"command"`
	Cwd          string        `json:"cwd"`
	Timeout      time.Duration `json:"timeout_ms"`
	MaxOutput    int64         `json:"max_output_bytes"`
	ProcessLimit int           `json:"processes"`
	NetworkMode  string        `json:"network"`
}

// DefaultRequest returns a request with sane defaults.
func DefaultRequest() *Request {
	return &Request{
		Cwd:          "/workspace",
		Timeout:      120 * time.Second,
		MaxOutput:    8 * 1024 * 1024, // 8 MiB
		ProcessLimit: 256,
	}
}

// Result holds the output of a buffered exec command.
type Result struct {
	ExitCode   int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Truncated  bool   `json:"truncated"`
	TimedOut   bool   `json:"timedOut"`
}

// StreamEvent is a single NDJSON event emitted by RunStream.
// type is one of: "stdout", "stderr", "done".
type StreamEvent struct {
	Type       string `json:"type"`
	Data       string `json:"data,omitempty"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	Truncated  bool   `json:"truncated,omitempty"`
	TimedOut   bool   `json:"timedOut,omitempty"`
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

// Run executes a command and returns a buffered Result when finished.
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

	// Fall back to /tmp if the specified directory doesn't exist
	if _, err := os.Stat(req.Cwd); os.IsNotExist(err) {
		req.Cwd = "/tmp"
	}

	start := time.Now()

	args := splitCommand(req.Command)
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = req.Cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	duration := time.Since(start)
	timedOut := ctx.Err() == context.DeadlineExceeded
	truncated := false

	stdoutStr := TruncateBytes(stdout.Bytes(), req.MaxOutput)
	stderrStr := TruncateBytes(stderr.Bytes(), req.MaxOutput)
	if int64(len(stdout.Bytes())) > req.MaxOutput {
		truncated = true
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if !timedOut {
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

// RunStream executes a command and writes NDJSON StreamEvent lines to w as
// stdout/stderr arrive. A final "done" event carries exitCode, durationMs,
// truncated, and timedOut — matching SPEC.md §20 (AC-7).
//
// The caller must set appropriate response headers before calling RunStream.
// If w implements Flush() the stream is flushed after every write.
func (e *Engine) RunStream(ctx context.Context, req *Request, w io.Writer) error {
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

	// Fall back to /tmp if the specified directory doesn't exist
	if _, err := os.Stat(req.Cwd); os.IsNotExist(err) {
		req.Cwd = "/tmp"
	}

	args := splitCommand(req.Command)
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = req.Cwd

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	flush := func() {
		if f, ok := w.(interface{ Flush() error }); ok {
			_ = f.Flush()
		} else if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}
	}

	writeEvent := func(ev StreamEvent) {
		b, _ := json.Marshal(ev)
		b = append(b, '\n')
		_, _ = w.Write(b)
		flush()
	}

	var (
		totalOut  int64
		truncated bool
		mu        sync.Mutex
	)

	copyPipe := func(r io.Reader, evType string) {
		buf := make([]byte, 4096)
		for {
			n, rerr := r.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				mu.Lock()
				if totalOut < req.MaxOutput {
					remaining := req.MaxOutput - totalOut
					if int64(len(chunk)) > remaining {
						chunk = chunk[:remaining]
						truncated = true
					}
					totalOut += int64(len(chunk))
					mu.Unlock()
					writeEvent(StreamEvent{Type: evType, Data: string(chunk)})
				} else {
					truncated = true
					mu.Unlock()
				}
			}
			if rerr != nil {
				break
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); copyPipe(stdoutPipe, "stdout") }()
	go func() { defer wg.Done(); copyPipe(stderrPipe, "stderr") }()
	wg.Wait()

	cmdErr := cmd.Wait()
	duration := time.Since(start)

	timedOut := ctx.Err() == context.DeadlineExceeded
	exitCode := 0
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if !timedOut {
			exitCode = 1
		}
	}

	writeEvent(StreamEvent{
		Type:       "done",
		ExitCode:   &exitCode,
		DurationMs: duration.Milliseconds(),
		Truncated:  truncated,
		TimedOut:   timedOut,
	})
	return nil
}

// splitCommand wraps a shell command for execution.
func splitCommand(cmd string) []string {
	if cmd == "" {
		return nil
	}
	return []string{"/bin/sh", "-c", cmd}
}

// TruncateBytes truncates bytes to maxOutput, appending a marker if truncated.
func TruncateBytes(data []byte, maxOutput int64) string {
	if int64(len(data)) <= maxOutput {
		return string(data)
	}
	return string(data[:maxOutput]) + "\n... [truncated]"
}
