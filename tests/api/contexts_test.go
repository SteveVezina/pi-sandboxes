package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/daemon"
)

func TestContextsListEndpoint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/contexts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["active"] == "" {
		t.Fatal("expected active context")
	}
	contexts, ok := body["contexts"].([]interface{})
	if !ok {
		t.Fatalf("expected contexts array, got %T", body["contexts"])
	}
	if len(contexts) == 0 {
		t.Fatal("expected at least local context")
	}
}

func TestContextsUseEndpoint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".pi"), 0o755); err != nil {
		t.Fatalf("create pi home: %v", err)
	}
	contextFile := []byte(`active_context: local
contexts:
  - name: remote-dev
    target: https://daemon.example.test
    transport: http
    auth:
      type: bearer-token
      token_env: PI_TEST_TOKEN
`)
	if err := os.WriteFile(filepath.Join(home, ".pi", "contexts.yaml"), contextFile, 0o600); err != nil {
		t.Fatalf("write context store: %v", err)
	}
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/v1/contexts/use", bytes.NewBufferString(`{"name":"remote-dev"}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["active"] != "remote-dev" {
		t.Fatalf("expected active remote-dev, got %v", body["active"])
	}
}
