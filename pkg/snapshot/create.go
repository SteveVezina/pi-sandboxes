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

// Create snapshots the workspace into the daemon-owned content-addressed
// store (~/.pi-box/snapshots/content-addressed-store, AC-13.4) and leaves
// a per-sandbox pointer (meta.json with the content hash). Identical
// workspace contents dedupe to one stored copy.
func (m *Manager) Create(name string, workspaceDir string) (*CreateResult, error) {
	staging, err := os.MkdirTemp("", "pi-snap-*")
	if err != nil {
		return &CreateResult{Name: name}, err
	}
	defer os.RemoveAll(staging)

	method, size, err := tryReflink(workspaceDir, staging)
	if err != nil {
		if method, size, err = tryTarCopy(workspaceDir, staging); err != nil {
			return &CreateResult{Name: name}, fmt.Errorf("snapshot copy failed: %w", err)
		}
	}

	hash, err := dirHash(staging)
	if err != nil {
		return &CreateResult{Name: name}, fmt.Errorf("hash snapshot: %w", err)
	}

	content := m.casContentDir(hash)
	if _, statErr := os.Stat(content); os.IsNotExist(statErr) {
		if err := os.MkdirAll(filepath.Dir(content), 0o755); err != nil {
			return &CreateResult{Name: name}, err
		}
		if err := os.Rename(staging, content); err != nil {
			// cross-device (staging in /tmp): fall back to a copy
			if _, cErr := copyDir(staging, content); cErr != nil {
				return &CreateResult{Name: name}, cErr
			}
		}
	}

	if _, err := m.createMeta(name, size, method, hash); err != nil {
		return &CreateResult{Name: name}, err
	}
	return &CreateResult{Name: name, Method: method, SizeBytes: size, Success: true}, nil
}

func (m *Manager) casContentDir(hash string) string {
	// m.snapshotsDir = <pi-home>/sandboxes/<id>/snapshots
	piHome := filepath.Dir(filepath.Dir(filepath.Dir(m.snapshotsDir)))
	return filepath.Join(piHome, "snapshots", "content-addressed-store", hash[:2], hash[2:])
}

// tryReflink copies src to dst file-by-file using copy-on-write clones
// (Linux FICLONE / APFS clonefile). It fails fast on the first file the
// filesystem can't clone (ext4, overlayfs) so the caller falls back to a
// plain copy.
func tryReflink(src, dst string) (string, int64, error) {
	if !reflinkSupported() {
		return "", 0, fmt.Errorf("reflink not supported on this platform")
	}
	srcInfo, err := os.Stat(src)
	if err != nil || !srcInfo.IsDir() {
		return "", 0, fmt.Errorf("reflink: source not a directory")
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return "", 0, err
	}

	var total int64
	err = filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if !info.Mode().IsRegular() {
			return nil // skip symlinks / devices — a snapshot of a workspace shouldn't have them
		}
		if err := cloneFile(p, target); err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		os.RemoveAll(dst)
		return "", 0, err
	}
	return "reflink", total, nil
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
