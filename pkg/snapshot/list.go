package snapshot

import (
	"fmt"
	"os"
	"sort"
	"time"
)

// SnapshotInfo holds summary info for a snapshot.
type SnapshotInfo struct {
	Name      string
	CreatedAt time.Time
	SizeBytes int64
	Method    string
}

// List returns all snapshots for this manager.
func (m *Manager) List() ([]SnapshotInfo, error) {
	entries, err := os.ReadDir(m.snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SnapshotInfo{}, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}

	var results []SnapshotInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		meta, err := m.GetMeta(entry.Name())
		if err != nil {
			continue
		}

		results = append(results, SnapshotInfo{
			Name:      meta.Name,
			CreatedAt: meta.CreatedAt,
			SizeBytes: meta.SizeBytes,
			Method:    meta.Method,
		})
	}

	// Sort by creation time, newest first
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	return results, nil
}
