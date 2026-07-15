package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Meta represents snapshot metadata.
type Meta struct {
	Name        string    `json:"name"`
	SandboxID   string    `json:"sandboxId"`
	CreatedAt   time.Time `json:"createdAt"`
	SizeBytes   int64     `json:"sizeBytes"`
	Method      string    `json:"method"` // "overlay", "reflink", "tar"
	WorkspaceID string    `json:"workspaceId"`
}

// Manager manages snapshots for a sandbox.
type Manager struct {
	sandboxID    string
	snapshotsDir string
}

// NewManager creates a snapshot manager for a sandbox.
func NewManager(sandboxID string) *Manager {
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	snapshotsDir := filepath.Join(home, ".pi-box", "sandboxes", sandboxID, "snapshots")
	return &Manager{
		sandboxID:    sandboxID,
		snapshotsDir: snapshotsDir,
	}
}

// EnsureDir creates the snapshots directory.
func (m *Manager) EnsureDir() error {
	return os.MkdirAll(m.snapshotsDir, 0755)
}

// CreateMeta writes snapshot metadata.
func (m *Manager) CreateMeta(name string, sizeBytes int64, method string) (*Meta, error) {
	if err := m.EnsureDir(); err != nil {
		return nil, fmt.Errorf("ensure snapshots dir: %w", err)
	}

	meta := &Meta{
		Name:        name,
		SandboxID:   m.sandboxID,
		CreatedAt:   time.Now().UTC(),
		SizeBytes:   sizeBytes,
		Method:      method,
		WorkspaceID: m.sandboxID,
	}

	metaPath := filepath.Join(m.snapshotsDir, name, "meta.json")
	dir := filepath.Join(m.snapshotsDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal meta: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write meta: %w", err)
	}

	return meta, nil
}

// GetMeta loads snapshot metadata.
func (m *Manager) GetMeta(name string) (*Meta, error) {
	metaPath := filepath.Join(m.snapshotsDir, name, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read meta: %w", err)
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal meta: %w", err)
	}

	return &meta, nil
}

// Exists checks if a snapshot exists.
func (m *Manager) Exists(name string) bool {
	_, err := m.GetMeta(name)
	return err == nil
}
