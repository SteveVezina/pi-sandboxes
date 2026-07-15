package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Store manages content-addressed snapshot storage.
type Store struct {
	rootDir string
}

// NewStore creates a new snapshot store.
func NewStore(rootDir string) *Store {
	return &Store{rootDir: rootDir}
}

// Create creates a content-addressed snapshot from a directory.
func (s *Store) Create(sandboxID, name string, srcDir string) (string, error) {
	// Create snapshot directory
	snapshotDir := s.snapshotDir(sandboxID, name)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return "", fmt.Errorf("create snapshot dir: %w", err)
	}

	// Calculate content hash of source directory
	hash, err := dirHash(srcDir)
	if err != nil {
		return "", fmt.Errorf("calculate hash: %w", err)
	}

	// Create content-addressed storage
	contentDir := s.contentDir(hash)
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return "", fmt.Errorf("create content dir: %w", err)
	}

	// Copy source to content-addressed storage
	if _, err := copyDir(srcDir, contentDir); err != nil {
		return "", fmt.Errorf("copy content: %w", err)
	}

	// Write snapshot metadata
	meta := SnapshotMeta{
		SandboxID: sandboxID,
		Name:      name,
		Hash:      hash,
		Content:   contentDir,
	}
	if err := meta.Write(snapshotDir); err != nil {
		return "", fmt.Errorf("write metadata: %w", err)
	}

	return snapshotDir, nil
}

// Get retrieves a snapshot by sandbox ID and name.
func (s *Store) Get(sandboxID, name string) (*SnapshotMeta, error) {
	snapshotDir := s.snapshotDir(sandboxID, name)
	return SnapshotMetaFromDir(snapshotDir)
}

// List returns all snapshots for a sandbox.
func (s *Store) List(sandboxID string) ([]SnapshotMeta, error) {
	sandboxSnapshotsDir := filepath.Join(s.rootDir, "snapshots", sandboxID)
	entries, err := os.ReadDir(sandboxSnapshotsDir)
	if err != nil {
		return nil, nil // No snapshots
	}

	var snapshots []SnapshotMeta
	for _, entry := range entries {
		if entry.IsDir() {
			meta, err := SnapshotMetaFromDir(filepath.Join(sandboxSnapshotsDir, entry.Name()))
			if err == nil {
				snapshots = append(snapshots, *meta)
			}
		}
	}
	return snapshots, nil
}

// Delete removes a snapshot.
func (s *Store) Delete(sandboxID, name string) error {
	snapshotDir := s.snapshotDir(sandboxID, name)
	return os.RemoveAll(snapshotDir)
}

// CleanupOrphans removes snapshots whose content is no longer referenced.
func (s *Store) CleanupOrphans() error {
	// In production, scan content directory and remove unreferenced content
	return nil
}

// snapshotDir returns the snapshot directory path.
func (s *Store) snapshotDir(sandboxID, name string) string {
	return filepath.Join(s.rootDir, "snapshots", sandboxID, name)
}

// contentDir returns the content-addressed storage directory.
func (s *Store) contentDir(hash string) string {
	return filepath.Join(s.rootDir, "content", hash[:2], hash[2:])
}

// dirHash calculates a SHA256 hash of a directory's contents.
func dirHash(dir string) (string, error) {
	hash := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err == nil {
				hash.Write(data)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
