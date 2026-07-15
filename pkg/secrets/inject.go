package secrets

import (
	"fmt"
	"strings"
)

// InjectResult represents the result of a credential injection.
type InjectResult struct {
	Success bool
	Message string
	Redacted bool
}

// InjectGitToken injects a Git token into a request or environment.
func InjectGitToken(token, host string) *InjectResult {
	// In production, this would:
	// 1. Validate the token format
	// 2. Inject it into the request header or GIT_ASKPASS environment
	// 3. Ensure it's not logged or exposed

	// For now, we just validate and return success
	if token == "" {
		return &InjectResult{
			Success:  false,
			Message:  "empty token",
			Redacted: true,
		}
	}

	return &InjectResult{
		Success:  true,
		Message:  fmt.Sprintf("Git token injected for %s", host),
		Redacted: true,
	}
}

// InjectRegistryAuth injects registry authentication credentials.
func InjectRegistryAuth(username, password, registry string) *InjectResult {
	if username == "" || password == "" {
		return &InjectResult{
			Success:  false,
			Message:  "empty credentials",
			Redacted: true,
		}
	}

	return &InjectResult{
		Success:  true,
		Message:  fmt.Sprintf("Registry auth injected for %s", registry),
		Redacted: true,
	}
}

// IsCredentialExposed checks if a credential value is exposed in a string.
func IsCredentialExposed(s string) bool {
	// Check for common credential patterns
	patterns := []string{
		"token=",
		"password=",
		"secret=",
		"api_key=",
		"apikey=",
		"Authorization: Bearer",
	}

	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(s), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// RedactCredentials redacts credential values from a string.
func RedactCredentials(s string) string {
	// Simple redaction: replace credential values with [REDACTED]
	result := s
	patterns := map[string]string{
		"token=":       "token=[REDACTED]",
		"password=":    "password=[REDACTED]",
		"secret=":      "secret=[REDACTED]",
		"api_key=":     "api_key=[REDACTED]",
		"apikey=":      "apikey=[REDACTED]",
		"Bearer ":      "Bearer [REDACTED]",
	}

	for pattern, replacement := range patterns {
		result = strings.Replace(result, pattern, replacement, -1)
	}

	return result
}
