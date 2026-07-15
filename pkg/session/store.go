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
	return filepath.Join(s.root, id)
}

// metaPath returns the path to meta.json for a session.
func (s *Store) metaPath(id string) string {
	return filepath.Join(s.sandboxDir(id), "meta.json")
}

// Create creates a new session and persists its metadata.
// Returns the session ID.
func (s *Store) Create(name, template, mode string) (string, error) {
	return s.CreateWithOptions(CreateOptions{
		Name: name, Template: template, Mode: mode,
	})
}

// CreateOptions contains optional metadata persisted at session creation.
type CreateOptions struct {
	Name          string
	Template      string
	Mode          string
	// RequestedMode preserves the mode the user asked for when selection
	// resolved a different backend (fallback visibility, SPEC §14.7.5).
	RequestedMode  string
	FallbackReason string
	Workspace      string
	WorkspaceMode  string
	TTL            int
}

// CreateWithOptions creates a new session and persists its metadata.
func (s *Store) CreateWithOptions(opts CreateOptions) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta := NewMeta(opts.Name, opts.Template, opts.Mode)
	meta.RequestedMode = opts.RequestedMode
	meta.FallbackReason = opts.FallbackReason
	if opts.Workspace != "" {
		meta.Workspace = opts.Workspace
	}
	if opts.WorkspaceMode != "" {
		meta.WorkspaceMode = opts.WorkspaceMode
	}
	if opts.TTL > 0 {
		meta.TTL = opts.TTL
	}

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

	entries, err := os.ReadDir(s.root)
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
		metaPath := filepath.Join(s.root, entry.Name(), "meta.json")
		if _, err := os.Stat(metaPath); err == nil {
			ids = append(ids, entry.Name())
		}
	}

	if ids == nil {
		ids = []string{}
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
