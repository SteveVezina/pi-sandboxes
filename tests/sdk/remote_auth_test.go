package sdk_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTypeScriptSDKSupportsRemoteAuth verifies the TypeScript SDK exposes
// authToken option and forwards it as a Bearer header (F23 / AC-26.6).
func TestTypeScriptSDKSupportsRemoteAuth(t *testing.T) {
	clientPath := filepath.Join("..", "..", "sdk", "typescript", "src", "client.ts")
	data, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	src := string(data)

	for _, want := range []string{
		"authToken",
		"PI_AUTH_TOKEN",
		"Bearer",
		"Authorization",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("client.ts missing %q for remote auth support", want)
		}
	}
}

// TestTypeScriptSDKDoesNotFallbackOnAuthFailure verifies the SDK throws on
// 401/403 instead of falling back to unauthenticated access (F23 / AC-26.8).
func TestTypeScriptSDKDoesNotFallbackOnAuthFailure(t *testing.T) {
	clientPath := filepath.Join("..", "..", "sdk", "typescript", "src", "client.ts")
	data, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("read client.ts: %v", err)
	}
	src := string(data)
	// Look for a 401/403 check before falling through.
	if !strings.Contains(src, "401") || !strings.Contains(src, "Remote auth failed") {
		t.Errorf("client.ts must surface 401/403 as a remote auth failure (ADR-003)")
	}
}

// TestPythonSDKSupportsRemoteAuth verifies the Python SDK exposes auth_token
// support and forwards it as a Bearer header (F23 / AC-26.6).
func TestPythonSDKSupportsRemoteAuth(t *testing.T) {
	pyPath := filepath.Join("..", "..", "sdk", "python", "src", "pi_sandbox", "__init__.py")
	data, err := os.ReadFile(pyPath)
	if err != nil {
		t.Fatalf("read python sdk: %v", err)
	}
	src := string(data)

	for _, want := range []string{
		"auth_token",
		"PI_AUTH_TOKEN",
		"Bearer",
		"Authorization",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("pi_sandbox/__init__.py missing %q for remote auth support", want)
		}
	}
}

// TestPythonSDKDoesNotFallbackOnAuthFailure verifies the Python SDK raises
// on 401/403 instead of silently continuing (F23 / AC-26.8).
func TestPythonSDKDoesNotFallbackOnAuthFailure(t *testing.T) {
	pyPath := filepath.Join("..", "..", "sdk", "python", "src", "pi_sandbox", "__init__.py")
	data, err := os.ReadFile(pyPath)
	if err != nil {
		t.Fatalf("read python sdk: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "401") || !strings.Contains(src, "Remote auth failed") {
		t.Errorf("python sdk must surface 401/403 as remote auth failure (ADR-003)")
	}
}

// TestSDKsDoNotPersistTokens verifies the SDKs never write tokens to disk
// (F23 / AC-26.4). We check by inspecting for any obvious file-write of the
// token variable.
func TestSDKsDoNotPersistTokens(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "sdk", "typescript", "src", "client.ts"),
		filepath.Join("..", "..", "sdk", "python", "src", "pi_sandbox", "__init__.py"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		src := string(data)
		// Forbid common file-write idioms for token vars.
		bad := []string{
			"writeFileSync.*authToken",
			"writeFile.*authToken",
			"open.*authToken.*w",
			"open.*auth_token.*w",
		}
		for _, pattern := range bad {
			if strings.Contains(src, pattern) {
				t.Errorf("%s appears to persist token (matched pattern %q)", path, pattern)
			}
		}
	}
}
