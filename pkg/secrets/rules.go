package secrets

import (
	"fmt"
	"sync"
)

// CredentialStore manages credential injection rules without storing plaintext secrets on disk.
type CredentialStore struct {
	mu       sync.RWMutex
	credentials map[string]Credential
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
			if h == host || (len(h) > 2 && h[0] == '*' && host[len(host)-len(h)+1:] == h[1:]) {
				creds = append(creds, c)
				break
			}
		}
	}
	return creds
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
