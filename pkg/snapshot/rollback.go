package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

// Rollback restores a workspace directory to a named snapshot, reading
// the content from the content-addressed store.
func (m *Manager) Rollback(name string, workspaceDir string) error {
	meta, err := m.GetMeta(name)
	if err != nil {
		return fmt.Errorf("snapshot %s not found", name)
	}

	var content string
	switch {
	case meta.Hash != "":
		content = m.casContentDir(meta.Hash)
	default:
		// legacy pre-CAS snapshot: content lived next to the meta
		content = filepath.Join(m.snapshotsDir, name)
	}
	if _, err := os.Stat(content); err != nil {
		return fmt.Errorf("snapshot content missing for %s", name)
	}

	if err := os.RemoveAll(workspaceDir); err != nil {
		return fmt.Errorf("clear workspace: %w", err)
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return fmt.Errorf("recreate workspace: %w", err)
	}
	_, err = copyDir(content, workspaceDir)
	return err
}
