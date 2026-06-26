package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

// Rollback restores a workspace to a snapshot.
func (m *Manager) Rollback(name string, workspaceDir string) error {
	snapDir := filepath.Join(m.snapshotsDir, name)

	// Check snapshot exists
	if _, err := os.Stat(snapDir); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", name)
	}

	// Check workspace exists
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		return fmt.Errorf("workspace %s not found", workspaceDir)
	}

	// Remove workspace contents
	if err := os.RemoveAll(workspaceDir); err != nil {
		return fmt.Errorf("clear workspace: %w", err)
	}

	// Copy snapshot contents to workspace
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("recreate workspace: %w", err)
	}

	_, err := copyDir(snapDir, workspaceDir)
	return err
}
