package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pictx "github.com/pi-sandbox/pi/pkg/context"
	"github.com/pi-sandbox/pi/pkg/remote"
)

// TestRemoteDaemon_EndToEndCreateExecDiff exercises the full remote workstation
// path described in F23 (T23.4 / AC-26.5):
//
//  1. The operator creates an http remote context with a bearer-token env var.
//  2. The pi-sandbox remote client authenticates every call.
//  3. The daemon's create/exec/diff routes are reached with valid auth.
//  4. The bearer token is never written into the contexts.yaml file (AC-26.4).
//  5. Auth failures surface as errors rather than falling back (AC-26.8).
func TestRemoteDaemon_EndToEndCreateExecDiff(t *testing.T) {
	const tokenVal = "supersecret-token-xyz"
	t.Setenv("PI_E2E_TOKEN", tokenVal)

	var got struct {
		createBody []byte
		execBody   []byte
		authHeader string
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		got.authHeader = r.Header.Get("Authorization")
		if got.authHeader != "Bearer "+tokenVal {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		got.createBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sb-1","name":"demo","template":"node-python","mode":"fast","state":"warm"}`))
	})
	mux.HandleFunc("/v1/sandboxes/sb-1/exec", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tokenVal {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		got.execBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"exitCode":0,"stdout":"ok\n","stderr":"","durationMs":12,"truncated":false,"timedOut":false}`))
	})
	mux.HandleFunc("/v1/sandboxes/sb-1/diff", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tokenVal {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"diff":"+ line\n"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Operator creates a remote context (stored in temp contexts.yaml).
	storePath := filepath.Join(t.TempDir(), "contexts.yaml")
	store, err := pictx.NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.Create(pictx.Context{
		Name:      "remote-ws",
		Target:    srv.URL,
		Transport: pictx.TransportHTTP,
		Auth:      pictx.AuthConfig{Type: pictx.AuthBearerToken, TokenEnv: "PI_E2E_TOKEN"},
	}); err != nil {
		t.Fatalf("Create context: %v", err)
	}
	if err := store.Use("remote-ws"); err != nil {
		t.Fatalf("Use: %v", err)
	}

	// AC-26.4: the contexts.yaml file must not contain the raw token.
	disk, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read contexts.yaml: %v", err)
	}
	if strings.Contains(string(disk), tokenVal) {
		t.Fatalf("contexts.yaml contains raw token; ADR-003 says tokens never persist to disk:\n%s", disk)
	}

	// Resolve and create a remote client.
	ctx, err := store.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	client, err := remote.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// 1. POST /v1/sandboxes (create) — authenticated.
	createReq := []byte(`{"name":"demo","template":"node-python","mode":"fast"}`)
	resp, err := client.Do("POST", "/v1/sandboxes", bytes.NewReader(createReq))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, body)
	}
	var created map[string]interface{}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created["id"] != "sb-1" {
		t.Fatalf("create response id = %v, want sb-1", created["id"])
	}
	if got.authHeader != "Bearer "+tokenVal {
		t.Fatalf("daemon saw auth header %q, want bearer %q", got.authHeader, tokenVal)
	}

	// 2. POST /v1/sandboxes/sb-1/exec — authenticated.
	execReq := []byte(`{"command":"echo ok","cwd":"/workspace"}`)
	resp, err = client.Do("POST", "/v1/sandboxes/sb-1/exec", bytes.NewReader(execReq))
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exec status = %d, body = %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), `"exitCode":0`) {
		t.Fatalf("exec response = %s", body)
	}

	// 3. GET /v1/sandboxes/sb-1/diff — authenticated.
	resp, err = client.Do("GET", "/v1/sandboxes/sb-1/diff", nil)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("diff status = %d, body = %s", resp.StatusCode, body)
	}
}

// TestRemoteDaemon_AuthFailureNeverFallsBack verifies AC-26.8: if the bearer
// token is wrong the remote client surfaces the 401 and does not silently
// proceed without auth.
func TestRemoteDaemon_AuthFailureNeverFallsBack(t *testing.T) {
	t.Setenv("PI_E2E_TOKEN", "wrong-token")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer srv.Close()

	ctx := pictx.Context{
		Name:      "remote-ws",
		Target:    srv.URL,
		Transport: pictx.TransportHTTP,
		Auth:      pictx.AuthConfig{Type: pictx.AuthBearerToken, TokenEnv: "PI_E2E_TOKEN"},
	}
	client, err := remote.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := client.Do("GET", "/v1/sandboxes", nil)
	if err != nil {
		// Acceptable: transport surfaces the auth failure directly.
		if !strings.Contains(strings.ToLower(err.Error()), "unauth") {
			t.Fatalf("error = %v, want 'unauth'-ish", err)
		}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no fallback)", resp.StatusCode)
	}
}

// TestRemoteDaemon_MissingTokenIsActionable verifies AC-26.1: remote API calls
// must be authenticated. If the bearer token env var is unset, NewClient must
// refuse to construct rather than letting an unauthenticated request fly.
func TestRemoteDaemon_MissingTokenIsActionable(t *testing.T) {
	_ = os.Unsetenv("PI_E2E_TOKEN_MISSING")
	ctx := pictx.Context{
		Name:      "remote-ws",
		Target:    "https://daemon:7777",
		Transport: pictx.TransportHTTP,
		Auth:      pictx.AuthConfig{Type: pictx.AuthBearerToken, TokenEnv: "PI_E2E_TOKEN_MISSING"},
	}
	if _, err := remote.NewClient(ctx); err == nil {
		t.Fatal("expected NewClient to fail when bearer token env var is unset")
	}
}
