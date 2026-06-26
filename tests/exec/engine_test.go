package exec_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/exec"
)

func engine() *exec.Engine {
	return exec.NewEngine(0)
}

func TestRun_Echo(t *testing.T) {
	result, err := engine().Run(context.Background(), &exec.Request{
		Command: "echo hello",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("Expected 'hello' in stdout, got '%s'", result.Stdout)
	}
	if result.DurationMs <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestRun_FailingCommand(t *testing.T) {
	result, err := engine().Run(context.Background(), &exec.Request{
		Command: "exit 42",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}
}

func TestRun_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := engine().Run(ctx, &exec.Request{
		Command: "sleep 10",
		Cwd:     "/tmp",
		Timeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !result.TimedOut {
		t.Error("Expected timedOut to be true")
	}
}

func TestRun_OutputTruncation(t *testing.T) {
	command := "python3 -c \"print('x' * 1048576)\""
	result, err := engine().Run(context.Background(), &exec.Request{
		Command:   command,
		Cwd:       "/tmp",
		Timeout:   5 * time.Second,
		MaxOutput: 100,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !result.Truncated {
		t.Error("Expected truncated to be true")
	}
	if len(result.Stdout) > 1024 {
		t.Errorf("Expected stdout <= 1024 bytes, got %d", len(result.Stdout))
	}
	if !strings.Contains(result.Stdout, "[truncated]") {
		t.Error("Expected truncation marker in stdout")
	}
}

func TestRun_EmptyCommand(t *testing.T) {
	_, err := engine().Run(context.Background(), &exec.Request{
		Command: "",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	})
	if err == nil {
		t.Fatal("Expected error for empty command, got nil")
	}
}

func TestRun_DefaultRequest(t *testing.T) {
	req := exec.DefaultRequest()
	if req.Cwd != "/workspace" {
		t.Errorf("Expected default CWD '/workspace', got '%s'", req.Cwd)
	}
	if req.Timeout != 120*time.Second {
		t.Errorf("Expected default timeout 120s, got %v", req.Timeout)
	}
	if req.MaxOutput != 8*1024*1024 {
		t.Errorf("Expected default maxOutput 8MiB, got %d", req.MaxOutput)
	}
}

func TestRun_StderrCaptured(t *testing.T) {
	result, err := engine().Run(context.Background(), &exec.Request{
		Command: "echo 'error message' >&2; exit 1",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(result.Stderr, "error message") {
		t.Errorf("Expected 'error message' in stderr, got '%s'", result.Stderr)
	}
}

func TestRun_StdoutAndStderrSeparate(t *testing.T) {
	result, err := engine().Run(context.Background(), &exec.Request{
		Command: "echo 'out'; echo 'err' >&2",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "out") {
		t.Errorf("Expected 'out' in stdout, got '%s'", result.Stdout)
	}
	if !strings.Contains(result.Stderr, "err") {
		t.Errorf("Expected 'err' in stderr, got '%s'", result.Stderr)
	}
}

func TestTruncateBytes_NoTruncation(t *testing.T) {
	result := exec.TruncateBytes([]byte("hello world"), 1024)
	if result != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", result)
	}
}

func TestTruncateBytes_Truncated(t *testing.T) {
	result := exec.TruncateBytes([]byte("hello world"), 5)
	if result != "hello\n... [truncated]" {
		t.Errorf("Expected truncated output, got '%s'", result)
	}
}
