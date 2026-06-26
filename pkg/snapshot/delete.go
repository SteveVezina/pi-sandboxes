package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

// Delete removes a snapshot.
func (m *Manager) Delete(name string) error {
	snapDir := filepath.Join(m.snapshotsDir, name)

	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", name)
	}

	return os.RemoveAll(snapDir)
}
