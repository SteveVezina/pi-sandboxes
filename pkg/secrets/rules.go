package secrets

import (
	"fmt"
	"strings"
	"sync"
)

// CredentialStore manages credential injection rules and holds resolved
// secret values in memory only — nothing is written to disk (ADR-006).
type CredentialStore struct {
	mu          sync.RWMutex
	credentials map[string]Credential
	values      map[string]string
}

// Credential represents a scoped credential.
type Credential struct {
	ID       string
	Name     string
	Type     string // git-token, registry-auth, etc.
	Hosts    []string
	InjectAs string // header, env, file
	Redacted bool
}

// NewCredentialStore creates a new credential store.
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{
		credentials: make(map[string]Credential),
		values:      make(map[string]string),
	}
}

// Add adds a credential to the store.
func (s *CredentialStore) Add(c Credential) error {
	if c.ID == "" {
		return fmt.Errorf("credential ID is required")
	}
	if len(c.Hosts) == 0 {
		return fmt.Errorf("credential must have at least one host")
	}
	if c.Type == "" {
		return fmt.Errorf("credential type is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[c.ID] = c
	return nil
}

// AddWithValue registers a credential rule together with its resolved
// secret value. The value is kept in memory only.
func (s *CredentialStore) AddWithValue(c Credential, value string) error {
	if value == "" {
		return fmt.Errorf("credential value is required")
	}
	if err := s.Add(c); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[c.ID] = value
	return nil
}

// Resolve returns the in-memory secret value for a credential ID.
func (s *CredentialStore) Resolve(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.values[id]
	if !ok {
		return "", fmt.Errorf("credential %s has no value", id)
	}
	return v, nil
}

// Remove deletes a credential rule and its value.
func (s *CredentialStore) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credentials, id)
	delete(s.values, id)
}

// Get retrieves a credential by ID.
func (s *CredentialStore) Get(id string) (*Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c, ok := s.credentials[id]
	if !ok {
		return nil, fmt.Errorf("credential %s not found", id)
	}
	return &c, nil
}

// List returns all credentials.
func (s *CredentialStore) List() []Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var creds []Credential
	for _, c := range s.credentials {
		creds = append(creds, c)
	}
	return creds
}

// GetForHost returns credentials that match a host.
func (s *CredentialStore) GetForHost(host string) []Credential {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var creds []Credential
	for _, c := range s.credentials {
		for _, h := range c.Hosts {
			if hostMatches(h, host) {
				creds = append(creds, c)
				break
			}
		}
	}
	return creds
}

// hostMatches reports whether pattern (exact host or "*.suffix") matches host.
func hostMatches(pattern, host string) bool {
	if pattern == host {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		return strings.HasSuffix(host, pattern[1:]) // ".suffix"
	}
	return false
}

// Redact redacts sensitive values from a string for logging.
func Redact(s string) string {
	// Simple redaction: replace alphanumeric sequences with asterisks
	// In production, this would use a more sophisticated approach
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		result[i] = '*'
	}
	return string(result)
}
