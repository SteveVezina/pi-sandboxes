package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateResult holds the result of a snapshot creation.
type CreateResult struct {
	Name      string `json:"name"`
	Method    string `json:"method"`
	SizeBytes int64  `json:"sizeBytes"`
	Success   bool   `json:"success"`
}

// Create creates a snapshot of the workspace.
func (m *Manager) Create(name string, workspaceDir string) (*CreateResult, error) {
	snapDir := filepath.Join(m.snapshotsDir, name)

	// Try reflink first (btrfs, xfs)
	method, size, err := tryReflink(workspaceDir, snapDir)
	if err == nil {
		m.CreateMeta(name, size, method)
		return &CreateResult{Name: name, Method: method, SizeBytes: size, Success: true}, nil
	}

	// Fall back to tar copy
	method, size, err = tryTarCopy(workspaceDir, snapDir)
	if err == nil {
		m.CreateMeta(name, size, method)
		return &CreateResult{Name: name, Method: method, SizeBytes: size, Success: true}, nil
	}

	return &CreateResult{Name: name, Success: false}, fmt.Errorf("snapshot creation failed: %w", err)
}

// tryReflink attempts reflink copy (fast, space-efficient).
func tryReflink(src, dst string) (string, int64, error) {
	// Check if source exists
	if _, err := os.Stat(src); err != nil {
		return "", 0, err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", 0, err
	}

	// Try reflink via cp --reflink=always
	// This is a stub — actual reflink requires syscall on Linux
	// For now, fall through to tar copy
	return "", 0, fmt.Errorf("reflink not supported on this platform")
}

// tryTarCopy creates a copy of the workspace.
func tryTarCopy(src, dst string) (string, int64, error) {
	if _, err := os.Stat(src); err != nil {
		return "", 0, err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return "", 0, err
	}

	// Copy directory recursively
	size, err := copyDir(src, dst)
	if err != nil {
		return "", 0, err
	}

	return "tar", size, nil
}

// copyDir copies a directory recursively, returning total size.
func copyDir(src, dst string) (int64, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !srcInfo.IsDir() {
		return 0, fmt.Errorf("src is not a directory")
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		info, err := entry.Info()
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}

		if entry.IsDir() {
			size, err := copyDir(srcPath, dstPath)
			if err != nil {
				return 0, err
			}
			totalSize += size
		} else {
			// Copy file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return 0, err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return 0, err
			}
		}
	}

	return totalSize, nil
}
