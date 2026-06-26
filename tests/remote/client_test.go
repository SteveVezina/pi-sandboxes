package remote_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	pictx "github.com/pi-sandbox/pi/pkg/context"
	"github.com/pi-sandbox/pi/pkg/remote"
)

func TestClient_UnixTransportPassesAuthNone(t *testing.T) {
	ctx := pictx.Context{
		Name: "local", Target: "unix:///tmp/sock", Transport: "unix",
		Auth: pictx.AuthConfig{Type: "none"},
	}
	c, err := remote.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.Transport() != "unix" {
		t.Fatalf("transport = %q, want unix", c.Transport())
	}
}

func TestClient_HTTPSendsBearerToken(t *testing.T) {
	t.Setenv("PI_TEST_TOKEN", "secret-xyz")
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ctx := pictx.Context{
		Name: "ws", Target: srv.URL, Transport: "http",
		Auth: pictx.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TEST_TOKEN"},
	}
	c, err := remote.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := c.Do("GET", "/v1/sandboxes", nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if sawAuth != "Bearer secret-xyz" {
		t.Fatalf("Authorization header = %q, want %q", sawAuth, "Bearer secret-xyz")
	}
	if !strings.Contains(string(body), "ok") {
		t.Fatalf("body = %q", body)
	}
}

func TestClient_HTTPMissingTokenIsActionableError(t *testing.T) {
	_ = os.Unsetenv("PI_TEST_MISSING_TOKEN")
	ctx := pictx.Context{
		Name: "ws", Target: "https://daemon:7777", Transport: "http",
		Auth: pictx.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TEST_MISSING_TOKEN"},
	}
	_, err := remote.NewClient(ctx)
	if err == nil {
		t.Fatal("expected error when bearer token env var is unset")
	}
	if !strings.Contains(err.Error(), "PI_TEST_MISSING_TOKEN") {
		t.Fatalf("error = %v, want mention of env var", err)
	}
}

func TestClient_RemoteAuthFailureDoesNotFallback(t *testing.T) {
	t.Setenv("PI_TEST_TOKEN", "wrong-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	ctx := pictx.Context{
		Name: "ws", Target: srv.URL, Transport: "http",
		Auth: pictx.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TEST_TOKEN"},
	}
	c, _ := remote.NewClient(ctx)

	resp, err := c.Do("GET", "/v1/sandboxes", nil)
	// Per ADR-003: never fall back to unauthenticated access; surface the 401.
	if err != nil {
		// If transport returns an error directly, that's also acceptable.
		if !strings.Contains(strings.ToLower(err.Error()), "unauth") {
			t.Fatalf("err = %v, want unauthorized-ish error", err)
		}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no fallback)", resp.StatusCode)
	}
}

func TestClient_SSHTransportSelected(t *testing.T) {
	ctx := pictx.Context{
		Name: "gpu-box", Target: "ssh://user@gpu-box.local", Transport: "ssh",
		Auth: pictx.AuthConfig{Type: "ssh-agent"},
	}
	c, err := remote.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.Transport() != "ssh" {
		t.Fatalf("transport = %q, want ssh", c.Transport())
	}
	// We don't actually dial in tests, but creation must succeed and
	// auth.type must be ssh-agent.
	if c.Auth() != "ssh-agent" {
		t.Fatalf("auth = %q, want ssh-agent", c.Auth())
	}
}

func TestClient_InvalidTransportRejected(t *testing.T) {
	ctx := pictx.Context{
		Name: "bad", Target: "x://y", Transport: "ftp",
		Auth: pictx.AuthConfig{Type: "none"},
	}
	_, err := remote.NewClient(ctx)
	if err == nil {
		t.Fatal("expected error for invalid transport")
	}
}

func TestSSHTransport_DoesNotRequireBearerToken(t *testing.T) {
	// SSH contexts must not need bearer-token env vars (auth flows through ssh-agent).
	ctx := pictx.Context{
		Name: "gpu-box", Target: "ssh://gpu-box.local", Transport: "ssh",
		Auth: pictx.AuthConfig{Type: "ssh-agent"},
	}
	c, err := remote.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.Auth() != "ssh-agent" {
		t.Fatalf("auth = %q, want ssh-agent", c.Auth())
	}
}
