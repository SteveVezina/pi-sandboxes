package microvm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrReseedFailed is returned when the reseed-on-restore hook fails.
var ErrReseedFailed = errors.New("microvm reseed hook failed")

// CreateWorkspaceDisk creates a writable ext4-formatted workspace disk file
// for the given sandbox under root. The size is in bytes.
//
// In this implementation the file is allocated as a sparse ext4 image
// placeholder. Actual mkfs.ext4 invocation happens in the linux backend.
func CreateWorkspaceDisk(root, sandboxID string, size int64) (Disk, error) {
	if root == "" {
		return Disk{}, fmt.Errorf("workspace disk root is required")
	}
	if sandboxID == "" {
		return Disk{}, fmt.Errorf("workspace disk sandbox id is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Disk{}, fmt.Errorf("create workspace disk root: %w", err)
	}
	path := filepath.Join(root, sandboxID+"-workspace.ext4")
	f, err := os.Create(path)
	if err != nil {
		return Disk{}, fmt.Errorf("create workspace disk: %w", err)
	}
	defer f.Close()
	if size > 0 {
		if err := f.Truncate(size); err != nil {
			return Disk{}, fmt.Errorf("size workspace disk: %w", err)
		}
	}
	return Disk{
		Path:       path,
		ReadOnly:   false,
		Filesystem: "ext4",
	}, nil
}

// TemplateRestore is the result of restoring a microVM template snapshot.
type TemplateRestore struct {
	Rootfs Disk
}

// WriteTemplateSnapshot writes a template snapshot to the given path. Used by
// tests and by template build pipelines.
func WriteTemplateSnapshot(path string, data []byte) error {
	if path == "" {
		return fmt.Errorf("template snapshot path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create template snapshot dir: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// RestoreTemplateSnapshot loads a read-only template snapshot into a TemplateRestore.
// Per ADR-001, the rootfs is always read-only.
func RestoreTemplateSnapshot(path string) (TemplateRestore, error) {
	if path == "" {
		return TemplateRestore{}, fmt.Errorf("template snapshot path is required")
	}
	if _, err := os.Stat(path); err != nil {
		return TemplateRestore{}, fmt.Errorf("stat template snapshot %s: %w", path, err)
	}
	return TemplateRestore{
		Rootfs: Disk{Path: path, ReadOnly: true},
	}, nil
}

// ReseedHook wraps a reseed callback so RunRestoreSequence can compose it
// with the readiness step in a single ordered call.
type ReseedHook func() error

// RunRestoreSequence runs the reseed hook then signals readiness. If the
// reseed hook returns an error, readiness is not signaled (ADR-001).
func RunRestoreSequence(reseed ReseedHook, ready func() error) error {
	if reseed != nil {
		if err := reseed(); err != nil {
			return fmt.Errorf("microvm restore: %w", err)
		}
	}
	if ready != nil {
		if err := ready(); err != nil {
			return fmt.Errorf("microvm ready: %w", err)
		}
	}
	return nil
}
