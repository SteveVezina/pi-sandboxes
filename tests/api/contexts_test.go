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
	if err := os.MkdirAll(filepath.Join(home, ".pi-box"), 0o755); err != nil {
		t.Fatalf("create Pi Box home: %v", err)
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
	if err := os.WriteFile(filepath.Join(home, ".pi-box", "contexts.yaml"), contextFile, 0o600); err != nil {
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

func TestContextCRUDEndpoints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	createBody := `{
		"name":"remote-dev",
		"target":"https://daemon.example.test",
		"transport":"http",
		"auth_type":"bearer-token",
		"token_env":"PI_REMOTE_TOKEN"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/contexts", bytes.NewBufferString(createBody))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/contexts/remote-dev", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var got map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got["token_env"] != "PI_REMOTE_TOKEN" {
		t.Fatalf("token_env = %v, want PI_REMOTE_TOKEN", got["token_env"])
	}

	updateBody := `{
		"name":"remote-dev",
		"target":"https://daemon2.example.test",
		"transport":"http",
		"auth_type":"bearer-token",
		"token_env":"PI_REMOTE_TOKEN_2"
	}`
	req = httptest.NewRequest(http.MethodPut, "/v1/contexts/remote-dev", bytes.NewBufferString(updateBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/contexts/remote-dev", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get after update expected 200, got %d: %s", w.Code, w.Body.String())
	}
	got = map[string]interface{}{}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode updated get: %v", err)
	}
	if got["target"] != "https://daemon2.example.test" || got["token_env"] != "PI_REMOTE_TOKEN_2" {
		t.Fatalf("updated context = %#v", got)
	}

	req = httptest.NewRequest(http.MethodDelete, "/v1/contexts/remote-dev", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/contexts/remote-dev", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get deleted expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContextDeleteActiveResetsToLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".pi-box"), 0o755); err != nil {
		t.Fatalf("create Pi Box home: %v", err)
	}
	contextFile := []byte(`active_context: remote-dev
contexts:
  - name: remote-dev
    target: https://daemon.example.test
    transport: http
    auth:
      type: bearer-token
      token_env: PI_TEST_TOKEN
`)
	if err := os.WriteFile(filepath.Join(home, ".pi-box", "contexts.yaml"), contextFile, 0o600); err != nil {
		t.Fatalf("write context store: %v", err)
	}
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodDelete, "/v1/contexts/remote-dev", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete active expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode delete: %v", err)
	}
	if body["active"] != "local" {
		t.Fatalf("active after delete = %v, want local", body["active"])
	}
}

func TestContextCRUDEndpointsProtectLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodDelete, "/v1/contexts/local", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete local expected 400, got %d: %s", w.Code, w.Body.String())
	}

	updateBody := `{"name":"local","target":"unix://x","transport":"unix","auth_type":"none"}`
	req = httptest.NewRequest(http.MethodPut, "/v1/contexts/local", bytes.NewBufferString(updateBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update local expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestContextCRUDEndpointsRejectRawCredentials(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	rawToken := `{
		"name":"bad-token",
		"target":"https://daemon.example.test",
		"transport":"http",
		"auth_type":"bearer-token",
		"token":"secret"
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/contexts", bytes.NewBufferString(rawToken))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("raw token expected 400, got %d: %s", w.Code, w.Body.String())
	}

	rawKey := `{
		"name":"bad-key",
		"target":"ssh://gpu-box.local",
		"transport":"ssh",
		"auth_type":"ssh-agent",
		"private_key":"-----BEGIN"
	}`
	req = httptest.NewRequest(http.MethodPost, "/v1/contexts", bytes.NewBufferString(rawKey))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("raw private key expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestActiveRemoteContextProxiesSandboxRoutes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_TEST_TOKEN", "secret-token")

	var sawAuth string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		if r.URL.RequestURI() != "/v1/sandboxes?state=warm" {
			t.Fatalf("remote path = %s, want /v1/sandboxes?state=warm", r.URL.RequestURI())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"remote-1","name":"from-remote","template":"base","mode":"fast","state":"WARM"}]`))
	}))
	defer remote.Close()

	if err := os.MkdirAll(filepath.Join(home, ".pi-box"), 0o755); err != nil {
		t.Fatalf("create Pi Box home: %v", err)
	}
	contextFile := []byte(`active_context: remote-dev
contexts:
  - name: remote-dev
    target: ` + remote.URL + `
    transport: http
    auth:
      type: bearer-token
      token_env: PI_TEST_TOKEN
`)
	if err := os.WriteFile(filepath.Join(home, ".pi-box", "contexts.yaml"), contextFile, 0o600); err != nil {
		t.Fatalf("write context store: %v", err)
	}

	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes?state=warm", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if sawAuth != "Bearer secret-token" {
		t.Fatalf("Authorization header = %q, want bearer token", sawAuth)
	}
	var sandboxes []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&sandboxes); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(sandboxes) != 1 || sandboxes[0]["name"] != "from-remote" {
		t.Fatalf("sandboxes = %#v, want remote response", sandboxes)
	}
}

func TestActiveRemoteContextDoesNotProxyContextRoutes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_TEST_TOKEN", "secret-token")

	remoteHit := false
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteHit = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer remote.Close()

	if err := os.MkdirAll(filepath.Join(home, ".pi-box"), 0o755); err != nil {
		t.Fatalf("create Pi Box home: %v", err)
	}
	contextFile := []byte(`active_context: remote-dev
contexts:
  - name: remote-dev
    target: ` + remote.URL + `
    transport: http
    auth:
      type: bearer-token
      token_env: PI_TEST_TOKEN
`)
	if err := os.WriteFile(filepath.Join(home, ".pi-box", "contexts.yaml"), contextFile, 0o600); err != nil {
		t.Fatalf("write context store: %v", err)
	}

	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/v1/contexts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if remoteHit {
		t.Fatal("context list should remain local and must not be proxied")
	}
}
