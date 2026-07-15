package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SnapshotMeta represents a snapshot's metadata.
type SnapshotMeta struct {
	SandboxID string    `json:"sandbox_id"`
	Name      string    `json:"name"`
	Hash      string    `json:"hash"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Write writes the snapshot metadata to a file.
func (m *SnapshotMeta) Write(dir string) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}

// SnapshotMetaFromDir reads snapshot metadata from a directory.
func SnapshotMetaFromDir(dir string) (*SnapshotMeta, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}

	var meta SnapshotMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &meta, nil
}
