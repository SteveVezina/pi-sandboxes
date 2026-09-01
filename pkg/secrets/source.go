package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValueSource describes where a credential's secret value comes from. The
// daemon resolves it once at registration; the value then lives only in
// the in-memory CredentialStore (ADR-006).
//
// Resolution precedence when more than one is set: Literal, then Env, then
// File. Keychain is not yet implemented.
type ValueSource struct {
	Literal  string `json:"value,omitempty"`
	Env      string `json:"env,omitempty"`      // daemon environment variable name
	File     string `json:"file,omitempty"`     // absolute path, must be outside ~/.pi-box
	Keychain string `json:"keychain,omitempty"` // reserved; not implemented
}

// Resolve returns the secret value for this source.
func (v ValueSource) Resolve() (string, error) {
	switch {
	case v.Literal != "":
		return v.Literal, nil
	case v.Env != "":
		val := os.Getenv(v.Env)
		if val == "" {
			return "", fmt.Errorf("daemon environment variable %q is empty or unset", v.Env)
		}
		return val, nil
	case v.File != "":
		if err := validateSecretFile(v.File); err != nil {
			return "", err
		}
		data, err := os.ReadFile(v.File)
		if err != nil {
			return "", fmt.Errorf("read secret file: %w", err)
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	case v.Keychain != "":
		return "", fmt.Errorf("keychain credential source is not implemented yet")
	default:
		return "", fmt.Errorf("credential value source is empty (want one of value, env, file)")
	}
}

// validateSecretFile enforces that a file-backed secret lives outside the
// Pi Box home, so credentials are never read from daemon-managed state.
func validateSecretFile(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("secret file path must be absolute: %q", path)
	}
	clean := filepath.Clean(path)
	home, err := os.UserHomeDir()
	if err == nil {
		piBox := filepath.Join(home, ".pi-box")
		if clean == piBox || strings.HasPrefix(clean, piBox+string(filepath.Separator)) {
			return fmt.Errorf("secret file must not live under %s", piBox)
		}
	}
	return nil
}
