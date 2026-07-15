package logs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single command execution log entry.
type Entry struct {
	Sequence   int       `json:"sequence"`
	Timestamp  time.Time `json:"timestamp"`
	Command    string    `json:"command"`
	ExitCode   int       `json:"exitCode"`
	DurationMs int64     `json:"durationMs"`
	TimedOut   bool      `json:"timedOut"`
	Truncated  bool      `json:"truncated"`
	StdoutPath string    `json:"stdoutPath"`
	StderrPath string    `json:"stderrPath"`
}

// Manager handles log storage for a sandbox.
type Manager struct {
	logsDir  string
	sequence int
}

// NewManager creates a log manager for the given sandbox ID. Logs live
// under the daemon-owned Pi Box home, next to the sandbox metadata.
func NewManager(sandboxID string) *Manager {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	logsDir := filepath.Join(home, ".pi-box", "sandboxes", sandboxID, "logs")
	m := &Manager{logsDir: logsDir}
	// Resume the sequence from what is already on disk so entries are
	// never overwritten across daemon restarts or manager instances.
	if matches, err := filepath.Glob(filepath.Join(logsDir, "exec-*.stdout")); err == nil {
		m.sequence = len(matches)
	}
	return m
}

// Dir returns the logs directory path.
func (m *Manager) Dir() string {
	return m.logsDir
}

// EnsureDir creates the logs directory if it doesn't exist.
func (m *Manager) EnsureDir() error {
	return os.MkdirAll(m.logsDir, 0755)
}

// Record saves a command execution log entry.
func (m *Manager) Record(command string, exitCode int, durationMs int64, timedOut, truncated bool, stdout, stderr string) (*Entry, error) {
	if err := m.EnsureDir(); err != nil {
		return nil, fmt.Errorf("ensure logs dir: %w", err)
	}

	m.sequence++
	seq := m.sequence

	// Write stdout to separate file
	stdoutPath := filepath.Join(m.logsDir, fmt.Sprintf("exec-%d.stdout", seq))
	if err := os.WriteFile(stdoutPath, []byte(stdout), 0644); err != nil {
		return nil, fmt.Errorf("write stdout: %w", err)
	}

	// Write stderr to separate file
	stderrPath := filepath.Join(m.logsDir, fmt.Sprintf("exec-%d.stderr", seq))
	if err := os.WriteFile(stderrPath, []byte(stderr), 0644); err != nil {
		return nil, fmt.Errorf("write stderr: %w", err)
	}

	// Create log entry
	entry := &Entry{
		Sequence:   seq,
		Timestamp:  time.Now().UTC(),
		Command:    command,
		ExitCode:   exitCode,
		DurationMs: durationMs,
		TimedOut:   timedOut,
		Truncated:  truncated,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
	}

	// Write JSON entry
	entryPath := filepath.Join(m.logsDir, fmt.Sprintf("exec-%d.json", seq))
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("marshal entry: %w", err)
	}

	if err := os.WriteFile(entryPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write entry: %w", err)
	}

	return entry, nil
}

// GetEntry loads a log entry by sequence number.
func (m *Manager) GetEntry(seq int) (*Entry, error) {
	entryPath := filepath.Join(m.logsDir, fmt.Sprintf("exec-%d.json", seq))
	data, err := os.ReadFile(entryPath)
	if err != nil {
		return nil, fmt.Errorf("read entry: %w", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal entry: %w", err)
	}

	return &entry, nil
}

// GetStdout returns the stdout content for a log entry.
func (m *Manager) GetStdout(seq int) (string, error) {
	data, err := os.ReadFile(filepath.Join(m.logsDir, fmt.Sprintf("exec-%d.stdout", seq)))
	return string(data), err
}

// GetStderr returns the stderr content for a log entry.
func (m *Manager) GetStderr(seq int) (string, error) {
	data, err := os.ReadFile(filepath.Join(m.logsDir, fmt.Sprintf("exec-%d.stderr", seq)))
	return string(data), err
}
