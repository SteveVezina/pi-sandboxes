package workspace

import (
	"fmt"
	"os"
)

// ReadFile reads a file from the workspace.
func (m *Manager) ReadFile(relPath string) ([]byte, error) {
	absPath, err := m.ValidatePath(relPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

// ReadFileText reads a file and returns it as a string.
func (m *Manager) ReadFileText(relPath string) (string, error) {
	data, err := m.ReadFile(relPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
