package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Mode describes how the workspace is mounted.
type Mode string

const (
	ModeCopy    Mode = "copy"    // Copy repo/files into sandbox
	ModeBind    Mode = "bind"    // Bind mount explicit host directory
	ModeOverlay Mode = "overlay" // Read-only base + writable upperdir
)

// Manager handles workspace operations for a sandbox session.
type Manager struct {
	workspaceDir string
	mode         Mode
}

// NewManager creates a workspace manager for the given sandbox ID.
func NewManager(sandboxID string, mode Mode) *Manager {
	workspaceDir := filepath.Join(os.TempDir(), "pi-sandbox-workspace", sandboxID)
	if mode == "" {
		mode = ModeCopy
	}
	return &Manager{
		workspaceDir: workspaceDir,
		mode:         mode,
	}
}

// Dir returns the workspace directory path.
func (m *Manager) Dir() string {
	return m.workspaceDir
}

// EnsureDir creates the workspace directory if it doesn't exist.
func (m *Manager) EnsureDir() error {
	return os.MkdirAll(m.workspaceDir, 0755)
}

// ValidatePath checks that a file path is within the workspace directory.
// Prevents path traversal attacks.
func (m *Manager) ValidatePath(relPath string) (string, error) {
	// Clean the path to prevent traversal
	cleaned := filepath.Clean(relPath)
	if strings.HasPrefix(cleaned, "..") || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("path traversal detected: %s", relPath)
	}
	absPath := filepath.Join(m.workspaceDir, cleaned)
	// Double-check the resolved path is within workspace
	if !strings.HasPrefix(absPath, m.workspaceDir+string(os.PathSeparator)) && absPath != m.workspaceDir {
		return "", fmt.Errorf("path traversal detected: %s", relPath)
	}
	return absPath, nil
}

// Mode returns the workspace mode.
func (m *Manager) Mode() Mode {
	return m.mode
}
