package logs_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/logs"
)

func TestRecord(t *testing.T) {
	m := newTestManager(t)

	entry, err := m.Record("echo hello", 0, 42, false, false, "hello\n", "err\n")
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if entry.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", entry.Sequence)
	}
	if entry.Command != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%s'", entry.Command)
	}
	if entry.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", entry.ExitCode)
	}
	if entry.DurationMs != 42 {
		t.Errorf("Expected duration 42ms, got %d", entry.DurationMs)
	}
	if entry.TimedOut {
		t.Error("Expected timedOut false")
	}
	if entry.Truncated {
		t.Error("Expected truncated false")
	}
}

func TestRecord_AutoIncrement(t *testing.T) {
	m := newTestManager(t)

	_, err := m.Record("cmd1", 0, 10, false, false, "out1", "err1")
	if err != nil {
		t.Fatalf("Record 1 failed: %v", err)
	}

	entry2, err := m.Record("cmd2", 1, 20, true, true, "out2", "err2")
	if err != nil {
		t.Fatalf("Record 2 failed: %v", err)
	}

	if entry2.Sequence != 2 {
		t.Errorf("Expected sequence 2, got %d", entry2.Sequence)
	}
}

func TestGetEntry(t *testing.T) {
	m := newTestManager(t)

	m.Record("test cmd", 0, 100, false, false, "stdout", "stderr")
	entry, err := m.GetEntry(1)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}

	if entry.Command != "test cmd" {
		t.Errorf("Expected 'test cmd', got '%s'", entry.Command)
	}
	if entry.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", entry.ExitCode)
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	m := newTestManager(t)

	_, err := m.GetEntry(99)
	if err == nil {
		t.Fatal("Expected error for nonexistent entry")
	}
}

func TestGetStdout(t *testing.T) {
	m := newTestManager(t)

	m.Record("test", 0, 10, false, false, "my stdout", "my stderr")
	stdout, err := m.GetStdout(1)
	if err != nil {
		t.Fatalf("GetStdout failed: %v", err)
	}
	if stdout != "my stdout" {
		t.Errorf("Expected 'my stdout', got '%s'", stdout)
	}
}

func TestGetStderr(t *testing.T) {
	m := newTestManager(t)

	m.Record("test", 0, 10, false, false, "my stdout", "my stderr")
	stderr, err := m.GetStderr(1)
	if err != nil {
		t.Fatalf("GetStderr failed: %v", err)
	}
	if stderr != "my stderr" {
		t.Errorf("Expected 'my stderr', got '%s'", stderr)
	}
}

func TestList(t *testing.T) {
	m := newTestManager(t)

	m.Record("cmd1", 0, 10, false, false, "out1", "err1")
	m.Record("cmd2", 1, 20, true, false, "out2", "err2")
	m.Record("cmd3", 0, 30, false, true, "out3", "err3")

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}

	// Should be newest first
	if entries[0].Sequence != 3 {
		t.Errorf("Expected first entry sequence 3, got %d", entries[0].Sequence)
	}
	if entries[2].Sequence != 1 {
		t.Errorf("Expected last entry sequence 1, got %d", entries[2].Sequence)
	}
}

func TestList_Empty(t *testing.T) {
	m := newTestManager(t)

	entries, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(entries))
	}
}

func TestHistory(t *testing.T) {
	m := newTestManager(t)

	m.Record("echo test", 0, 50, false, false, "out", "err")
	m.Record("exit 1", 1, 10, false, false, "", "error")

	history, err := m.History()
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(history))
	}

	if history[0].Command != "exit 1" {
		t.Errorf("Expected first history entry 'exit 1', got '%s'", history[0].Command)
	}
	if history[0].ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", history[0].ExitCode)
	}
}

func TestHistory_Empty(t *testing.T) {
	m := newTestManager(t)

	history, err := m.History()
	if err != nil {
		t.Fatalf("History failed: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected 0 history entries, got %d", len(history))
	}
}

func TestCount(t *testing.T) {
	m := newTestManager(t)

	if m.Count() != 0 {
		t.Error("Expected 0 entries before recording")
	}

	m.Record("cmd1", 0, 10, false, false, "out", "err")
	if m.Count() != 1 {
		t.Errorf("Expected 1 entry, got %d", m.Count())
	}

	m.Record("cmd2", 0, 20, false, false, "out", "err")
	if m.Count() != 2 {
		t.Errorf("Expected 2 entries, got %d", m.Count())
	}
}

func TestEnsureDir(t *testing.T) {
	m := newTestManager(t)

	if err := m.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if _, err := os.Stat(m.Dir()); os.IsNotExist(err) {
		t.Fatal("Expected logs dir to exist")
	}
}

func TestLogFilesCreated(t *testing.T) {
	m := newTestManager(t)

	m.Record("test", 0, 10, false, false, "stdout data", "stderr data")

	// Check stdout file exists
	stdoutPath := filepath.Join(m.Dir(), "exec-1.stdout")
	if _, err := os.Stat(stdoutPath); os.IsNotExist(err) {
		t.Fatal("Expected stdout file to exist")
	}

	// Check stderr file exists
	stderrPath := filepath.Join(m.Dir(), "exec-1.stderr")
	if _, err := os.Stat(stderrPath); os.IsNotExist(err) {
		t.Fatal("Expected stderr file to exist")
	}

	// Check JSON entry exists
	jsonPath := filepath.Join(m.Dir(), "exec-1.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatal("Expected JSON entry to exist")
	}
}

func TestTimedOutFlag(t *testing.T) {
	m := newTestManager(t)

	entry, err := m.Record("sleep 10", 137, 5000, true, false, "", "")
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if !entry.TimedOut {
		t.Error("Expected timedOut true")
	}
}

func TestTruncatedFlag(t *testing.T) {
	m := newTestManager(t)

	entry, err := m.Record("big output", 0, 100, false, true, "truncated", "")
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if !entry.Truncated {
		t.Error("Expected truncated true")
	}
}

func TestRecordEgress_RoundTripNewestFirst(t *testing.T) {
	m := newTestManager(t)

	if err := m.RecordEgress("evil.example.com", false, "not allowlisted"); err != nil {
		t.Fatalf("RecordEgress: %v", err)
	}
	if err := m.RecordEgress("169.254.169.254", false, "not allowlisted"); err != nil {
		t.Fatalf("RecordEgress: %v", err)
	}

	events, err := m.EgressEvents()
	if err != nil {
		t.Fatalf("EgressEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].Host != "169.254.169.254" {
		t.Errorf("want newest-first ordering, got %q first", events[0].Host)
	}
	if events[0].Allowed {
		t.Error("recorded event should be a denial")
	}
	if events[1].Host != "evil.example.com" {
		t.Errorf("second event host = %q", events[1].Host)
	}
}

func TestEgressEvents_EmptyWhenNoLog(t *testing.T) {
	m := newTestManager(t)
	events, err := m.EgressEvents()
	if err != nil {
		t.Fatalf("EgressEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("want empty, got %d", len(events))
	}
}

func TestTimestamp(t *testing.T) {
	m := newTestManager(t)

	before := time.Now().UTC()
	m.Record("test", 0, 10, false, false, "out", "err")
	after := time.Now().UTC()

	entry, err := m.GetEntry(1)
	if err != nil {
		t.Fatalf("GetEntry failed: %v", err)
	}

	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("Timestamp %v not between %v and %v", entry.Timestamp, before, after)
	}
}

// newTestManager isolates NewManager's ~/.pi-box/sandboxes/<id>/logs join
// to a scratch HOME so tests never write into the real user's Pi Box home
// (NewManager takes a sandbox ID, not a directory — passing a full tmp
// path here would leak files under ~/.pi-box/sandboxes/<tmp-path>/logs).
func newTestManager(t *testing.T) *logs.Manager {
	t.Setenv("HOME", t.TempDir())
	return logs.NewManager("test-sandbox-" + randomID())
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
