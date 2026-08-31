package logs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EgressEvent is one egress-proxy decision recorded for a sandbox
// (ADR-006 / F30 T30.6). It never carries credential material.
type EgressEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Host      string    `json:"host"`
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason"`
}

func (m *Manager) egressPath() string {
	return filepath.Join(m.logsDir, "egress.jsonl")
}

// RecordEgress appends one egress decision to the sandbox's egress log.
// Callers record denials here; allow decisions stay at debug level in the
// daemon log only.
func (m *Manager) RecordEgress(host string, allowed bool, reason string) error {
	if err := m.EnsureDir(); err != nil {
		return fmt.Errorf("ensure logs dir: %w", err)
	}
	line, err := json.Marshal(EgressEvent{
		Timestamp: time.Now().UTC(),
		Host:      host,
		Allowed:   allowed,
		Reason:    reason,
	})
	if err != nil {
		return fmt.Errorf("marshal egress event: %w", err)
	}
	f, err := os.OpenFile(m.egressPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open egress log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write egress event: %w", err)
	}
	return nil
}

// EgressEvents returns all recorded egress decisions, newest first.
func (m *Manager) EgressEvents() ([]EgressEvent, error) {
	f, err := os.Open(m.egressPath())
	if err != nil {
		if os.IsNotExist(err) {
			return []EgressEvent{}, nil
		}
		return nil, fmt.Errorf("open egress log: %w", err)
	}
	defer f.Close()

	var events []EgressEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e EgressEvent
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan egress log: %w", err)
	}

	// newest first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events, nil
}
