package remote

import (
	"fmt"
	"os"

	pictx "github.com/pi-sandbox/pi/pkg/context"
)

// resolveBearerToken loads the bearer token from the env var referenced by
// the auth config. Per ADR-003, tokens are never persisted to disk inside
// sandbox workspaces or context files; they live in process environment only.
func resolveBearerToken(auth pictx.AuthConfig) (string, error) {
	if auth.Type != pictx.AuthBearerToken {
		return "", fmt.Errorf("resolveBearerToken: auth.type %q is not bearer-token", auth.Type)
	}
	if auth.TokenEnv == "" {
		return "", fmt.Errorf("bearer-token auth requires auth.token_env")
	}
	tok := os.Getenv(auth.TokenEnv)
	if tok == "" {
		return "", fmt.Errorf("bearer token env var %s is not set; remote auth requires a token (no unauthenticated fallback)", auth.TokenEnv)
	}
	return tok, nil
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return ""
}
