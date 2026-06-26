package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store manages session metadata persistence.
type Store struct {
	root string
	mu   sync.RWMutex
}

// NewStore creates a new session store at the given root directory.
func NewStore(root string) *Store {
	return &Store{root: root}
}

// sandboxDir returns the path for a session's metadata directory.
func (s *Store) sandboxDir(id string) string {
	return filepath.Join(s.root, "sandboxes", id)
}

// metaPath returns the path to meta.json for a session.
func (s *Store) metaPath(id string) string {
	return filepath.Join(s.sandboxDir(id), "meta.json")
}

// Create creates a new session and persists its metadata.
// Returns the session ID.
func (s *Store) Create(name, template, mode string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta := NewMeta(name, template, mode)

	dir := s.sandboxDir(meta.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create session dir: %w", err)
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(s.metaPath(meta.ID), data, 0644); err != nil {
		return "", fmt.Errorf("write metadata: %w", err)
	}

	return meta.ID, nil
}

// Get retrieves session metadata by ID.
func (s *Store) Get(id string) (*Meta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metaPath := s.metaPath(id)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("session %s not found: %w", id, err)
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// UpdateState updates the state of a session.
func (s *Store) UpdateState(id string, state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, err := s.load(id)
	if err != nil {
		return err
	}

	meta.State = state
	meta.UpdatedAt = time.Now()

	return s.save(id, meta)
}

// UpdateTTL updates the TTL for a session.
func (s *Store) UpdateTTL(id string, ttl int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, err := s.load(id)
	if err != nil {
		return err
	}

	meta.TTL = ttl
	meta.UpdatedAt = time.Now()

	return s.save(id, meta)
}

// UpdateLastUsed updates the last used timestamp for TTL calculation.
func (s *Store) UpdateLastUsed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, err := s.load(id)
	if err != nil {
		return err
	}

	meta.LastUsedAt = time.Now()
	meta.UpdatedAt = time.Now()

	return s.save(id, meta)
}

// List returns all session IDs.
func (s *Store) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sandboxesDir := filepath.Join(s.root, "sandboxes")
	entries, err := os.ReadDir(sandboxesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read sandboxes dir: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(sandboxesDir, entry.Name(), "meta.json")
		if _, err := os.Stat(metaPath); err == nil {
			ids = append(ids, entry.Name())
		}
	}

	return ids, nil
}

// Delete removes a session's metadata directory.
// This is idempotent — calling on a non-existent session returns nil.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.sandboxDir(id)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // idempotent
	}

	return os.RemoveAll(dir)
}

// load reads and parses metadata for a session. Caller must hold lock.
func (s *Store) load(id string) (*Meta, error) {
	data, err := os.ReadFile(s.metaPath(id))
	if err != nil {
		return nil, fmt.Errorf("session %s not found: %w", id, err)
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &meta, nil
}

// save writes metadata to disk. Caller must hold lock.
func (s *Store) save(id string, meta *Meta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	return os.WriteFile(s.metaPath(id), data, 0644)
}
