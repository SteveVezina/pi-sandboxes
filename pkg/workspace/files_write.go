package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes content to a file in the workspace.
// Creates parent directories if they don't exist.
func (m *Manager) WriteFile(relPath string, data []byte) error {
	absPath, err := m.ValidatePath(relPath)
	if err != nil {
		return err
	}

	// Create parent directories
	parent := filepath.Dir(absPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("create parent directories: %w", err)
	}

	// Write file
	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// WriteFileText writes a string to a file.
func (m *Manager) WriteFileText(relPath string, content string) error {
	return m.WriteFile(relPath, []byte(content))
}
