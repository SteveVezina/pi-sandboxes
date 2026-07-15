//go:build linux
// +build linux

package microvm

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"
)

// ExecResult holds the result of a command execution.
type ExecResult struct {
	ExitCode   int
	Stdout     []byte
	Stderr     []byte
	DurationMs int64
	TimedOut   bool
	Truncated  bool
}

// ExecConfig holds configuration for command execution.
type ExecConfig struct {
	Command        string
	Cwd            string
	TimeoutMs      int64
	MaxOutputBytes int64
}

// Exec runs a command inside the guest via the vsock control plane.
func (c *Client) Exec(ctx context.Context, config ExecConfig) (*ExecResult, error) {
	start := time.Now()

	// Create exec request frame
	reqFrame, err := NewExecRequestFrame(
		fmt.Sprintf("exec-%d", start.UnixNano()),
		c.sandboxID,
		ExecRequestPayload{
			Command:        config.Command,
			Cwd:            config.Cwd,
			TimeoutMs:      config.TimeoutMs,
			MaxOutputBytes: config.MaxOutputBytes,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create exec request: %w", err)
	}

	if _, err := c.Send(reqFrame); err != nil {
		return nil, fmt.Errorf("exec request failed: %w", err)
	}

	// Collect stream frames
	var stdout, stderr bytes.Buffer
	var truncated bool

	done := make(chan struct{})
	go func() {
		defer close(done)
		streamCh, errCh := c.StreamFrames()
		for {
			select {
			case frame, ok := <-streamCh:
				if !ok {
					return
				}
				stream, data, err := DecodeStreamPayload(frame)
				if err != nil {
					return
				}
				if config.MaxOutputBytes > 0 && (stdout.Len()+stderr.Len()) >= int(config.MaxOutputBytes) {
					truncated = true
					return
				}
				if stream == "stdout" {
					stdout.Write(data)
				} else {
					stderr.Write(data)
				}
			case <-errCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for response or timeout
	select {
	case <-done:
		// Read final response
		finalResp, err := c.Send(Frame{
			Type:      FrameTypeRequest,
			ID:        fmt.Sprintf("exec-resp-%d", start.UnixNano()),
			SandboxID: c.sandboxID,
			Method:    "exec",
		})
		if err != nil {
			// No final response, return what we have
			return &ExecResult{
				ExitCode:   -1,
				Stdout:     stdout.Bytes(),
				Stderr:     stderr.Bytes(),
				DurationMs: time.Since(start).Milliseconds(),
				Truncated:  truncated,
			}, nil
		}

		payload, err := ExecFramePayload(finalResp)
		if err != nil {
			return nil, fmt.Errorf("decode exec result: %w", err)
		}

		return &ExecResult{
			ExitCode:   payload.ExitCode,
			Stdout:     stdout.Bytes(),
			Stderr:     stderr.Bytes(),
			DurationMs: payload.DurationMs,
			TimedOut:   payload.TimedOut,
			Truncated:  truncated,
		}, nil
	case <-ctx.Done():
		return &ExecResult{
			ExitCode:   -1,
			Stdout:     stdout.Bytes(),
			Stderr:     stderr.Bytes(),
			DurationMs: time.Since(start).Milliseconds(),
			TimedOut:   true,
			Truncated:  truncated,
		}, ctx.Err()
	}
}

// ExecStreaming runs a command and returns channels for stdout, stderr, and result.
func (c *Client) ExecStreaming(ctx context.Context, config ExecConfig) (<-chan []byte, <-chan []byte, <-chan *ExecResult, <-chan error) {
	stdoutCh := make(chan []byte, 100)
	stderrCh := make(chan []byte, 100)
	resultCh := make(chan *ExecResult, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(stdoutCh)
		defer close(stderrCh)
		defer close(resultCh)
		defer close(errCh)

		start := time.Now()
		var stdout, stderr bytes.Buffer
		var mu sync.Mutex
		var truncated bool

		// Create exec request
		reqFrame, err := NewExecRequestFrame(
			fmt.Sprintf("exec-stream-%d", start.UnixNano()),
			c.sandboxID,
			ExecRequestPayload{
				Command:        config.Command,
				Cwd:            config.Cwd,
				TimeoutMs:      config.TimeoutMs,
				MaxOutputBytes: config.MaxOutputBytes,
			},
		)
		if err != nil {
			errCh <- fmt.Errorf("create exec request: %w", err)
			return
		}

		if _, err := c.Send(reqFrame); err != nil {
			errCh <- fmt.Errorf("exec request failed: %w", err)
			return
		}

		// Stream frames
		streamCh, streamErrCh := c.StreamFrames()
		for {
			select {
			case frame, ok := <-streamCh:
				if !ok {
					goto done
				}
				stream, data, err := DecodeStreamPayload(frame)
				if err != nil {
					goto done
				}
				mu.Lock()
				if config.MaxOutputBytes > 0 && (stdout.Len()+stderr.Len()) >= int(config.MaxOutputBytes) {
					truncated = true
					mu.Unlock()
					goto done
				}
				if stream == "stdout" {
					stdout.Write(data)
					stdoutCh <- data
				} else {
					stderr.Write(data)
					stderrCh <- data
				}
				mu.Unlock()
			case <-streamErrCh:
				goto done
			case <-ctx.Done():
				goto done
			}
		}

	done:
		// Final response
		finalResp, err := c.Send(Frame{
			Type:      FrameTypeRequest,
			ID:        fmt.Sprintf("exec-stream-resp-%d", start.UnixNano()),
			SandboxID: c.sandboxID,
			Method:    "exec",
		})

		result := &ExecResult{
			ExitCode:   -1,
			Stdout:     stdout.Bytes(),
			Stderr:     stderr.Bytes(),
			DurationMs: time.Since(start).Milliseconds(),
			Truncated:  truncated,
		}

		if err == nil {
			if payload, err := ExecFramePayload(finalResp); err == nil {
				result.ExitCode = payload.ExitCode
				result.DurationMs = payload.DurationMs
				result.TimedOut = payload.TimedOut
			}
		}

		resultCh <- result
	}()

	return stdoutCh, stderrCh, resultCh, errCh
}
