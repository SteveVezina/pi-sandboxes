package secrets

import (
	"os"
	"path/filepath"
	"strings"
)

// TokenHelper manages token-based credentials for Git.
type TokenHelper struct {
	Enabled bool
	Scopes  []string // Which hosts this token is for
	Helper  string   // Path to credential helper
}

// Setup creates the credential helper script for Git.
func (t *TokenHelper) Setup(broker *Broker) error {
	if !t.Enabled {
		return nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	dir := filepath.Join(home, ".pi-box", "secrets")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return nil
}

// IsScoped checks if a URL is within the token's scopes.
func (t *TokenHelper) IsScoped(url string) bool {
	if !t.Enabled {
		return false
	}
	for _, scope := range t.Scopes {
		if strings.Contains(url, scope) {
			return true
		}
	}
	return len(t.Scopes) == 0 // Empty scopes = all hosts
}
