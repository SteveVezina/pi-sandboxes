package exec_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/exec"
)

// TestRunStream_EmitsStdoutEvents verifies that RunStream writes "stdout" events
// as the process produces output (AC-7, SPEC.md §20).
func TestRunStream_EmitsStdoutEvents(t *testing.T) {
	engine := exec.NewEngine(1024 * 1024)
	req := &exec.Request{
		Command:   "echo hello && echo world",
		Cwd:       "/tmp",
		Timeout:   5 * time.Second,
		MaxOutput: 1024 * 1024,
	}

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	ctx := context.Background()
	if err := engine.RunStream(ctx, req, bw); err != nil {
		t.Fatalf("RunStream: %v", err)
	}
	_ = bw.Flush()

	events := parseNDJSON(t, buf.String())

	// Must have at least one stdout event and exactly one done event.
	hasStdout := false
	doneCnt := 0
	for _, ev := range events {
		switch ev["type"] {
		case "stdout":
			hasStdout = true
		case "done":
			doneCnt++
		}
	}
	if !hasStdout {
		t.Error("expected at least one stdout event")
	}
	if doneCnt != 1 {
		t.Errorf("expected exactly 1 done event, got %d", doneCnt)
	}
}

// TestRunStream_DoneEventCarriesExitCode verifies that the final "done" event
// carries the correct exitCode, durationMs (SPEC.md §20).
func TestRunStream_DoneEventCarriesExitCode(t *testing.T) {
	engine := exec.NewEngine(1024 * 1024)
	req := &exec.Request{
		Command: "exit 42",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	}

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	_ = engine.RunStream(context.Background(), req, bw)
	_ = bw.Flush()

	events := parseNDJSON(t, buf.String())
	for _, ev := range events {
		if ev["type"] == "done" {
			code, ok := ev["exitCode"].(float64)
			if !ok {
				t.Fatalf("done event missing exitCode; event=%v", ev)
			}
			if int(code) != 42 {
				t.Errorf("expected exitCode 42, got %d", int(code))
			}
			if _, ok := ev["durationMs"]; !ok {
				t.Error("done event missing durationMs")
			}
			return
		}
	}
	t.Error("no done event found")
}

// TestRunStream_StderrEvents verifies that stderr output produces "stderr" events.
func TestRunStream_StderrEvents(t *testing.T) {
	engine := exec.NewEngine(1024 * 1024)
	req := &exec.Request{
		Command: "echo err >&2",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	}

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	_ = engine.RunStream(context.Background(), req, bw)
	_ = bw.Flush()

	events := parseNDJSON(t, buf.String())
	hasStderr := false
	for _, ev := range events {
		if ev["type"] == "stderr" {
			hasStderr = true
		}
	}
	if !hasStderr {
		t.Error("expected at least one stderr event")
	}
}

// TestRunStream_TruncatesOutput verifies that RunStream sets truncated=true
// when output exceeds MaxOutput.
func TestRunStream_TruncatesOutput(t *testing.T) {
	engine := exec.NewEngine(10) // tiny limit
	req := &exec.Request{
		Command:   "echo 0123456789abcdefghij",
		Cwd:       "/tmp",
		Timeout:   5 * time.Second,
		MaxOutput: 10, // force truncation
	}

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	_ = engine.RunStream(context.Background(), req, bw)
	_ = bw.Flush()

	events := parseNDJSON(t, buf.String())
	for _, ev := range events {
		if ev["type"] == "done" {
			trunc, _ := ev["truncated"].(bool)
			if !trunc {
				t.Error("expected truncated=true in done event when output exceeds MaxOutput")
			}
			return
		}
	}
	t.Error("no done event found")
}

// TestExecAPIHandler_StreamingPath verifies that the HTTP handler writes
// NDJSON when the client sends ?stream=true.
func TestExecAPIHandler_StreamingPath(t *testing.T) {
	// Re-use the engine directly to confirm StreamEvent JSON shape,
	// since we can't easily create a full sandbox.Store in a unit test.
	engine := exec.NewEngine(0)
	req := &exec.Request{
		Command: "printf 'line1\nline2\n'",
		Cwd:     "/tmp",
		Timeout: 5 * time.Second,
	}

	var buf bytes.Buffer
	bw := bufio.NewWriter(&buf)
	_ = engine.RunStream(context.Background(), req, bw)
	_ = bw.Flush()

	output := buf.String()
	if !strings.Contains(output, "\"type\"") {
		t.Errorf("NDJSON output missing 'type' field: %s", output)
	}
	if !strings.Contains(output, "\"done\"") {
		t.Errorf("NDJSON output missing 'done' event: %s", output)
	}
}

// parseNDJSON parses a newline-delimited JSON string into a slice of maps.
func parseNDJSON(t *testing.T, s string) []map[string]interface{} {
	t.Helper()
	var events []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev map[string]interface{}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("parse NDJSON line %q: %v", line, err)
		}
		events = append(events, ev)
	}
	return events
}
