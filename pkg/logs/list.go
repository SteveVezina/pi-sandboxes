package logs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// HistoryEntry is a summary of a log entry for history display.
type HistoryEntry struct {
	Sequence   int    `json:"sequence"`
	Timestamp  string `json:"timestamp"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
	TimedOut   bool   `json:"timedOut"`
	Truncated  bool   `json:"truncated"`
}

// List returns all log entries sorted by sequence (newest first).
func (m *Manager) List() ([]Entry, error) {
	entries, err := os.ReadDir(m.logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read logs dir: %w", err)
	}

	var result []Entry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(m.logsDir, entry.Name()))
		if err != nil {
			continue
		}

		var e Entry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}

		result = append(result, e)
	}

	// Sort by sequence descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Sequence > result[j].Sequence
	})

	return result, nil
}

// History returns a summary of all log entries for history display.
func (m *Manager) History() ([]HistoryEntry, error) {
	entries, err := m.List()
	if err != nil {
		return nil, err
	}

	var history []HistoryEntry
	for _, e := range entries {
		history = append(history, HistoryEntry{
			Sequence:   e.Sequence,
			Timestamp:  e.Timestamp.Format(time.RFC3339),
			Command:    e.Command,
			ExitCode:   e.ExitCode,
			DurationMs: e.DurationMs,
			TimedOut:   e.TimedOut,
			Truncated:  e.Truncated,
		})
	}

	return history, nil
}

// Count returns the number of log entries.
func (m *Manager) Count() int {
	entries, err := os.ReadDir(m.logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}
	return count
}
