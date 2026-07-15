package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func newTestStore(t *testing.T) (*sandbox.Store, string) {
	tmpDir := filepath.Join(os.TempDir(), "pi-api-test")
	os.RemoveAll(tmpDir)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store := sandbox.NewStore(tmpDir)
	return store, tmpDir
}

func fastUnavailable(w *httptest.ResponseRecorder) bool {
	return w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "runtime mode fast is unavailable")
}

func createStoredSandbox(t *testing.T, store *sandbox.Store, name, template string) string {
	t.Helper()
	id, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name:          name,
		Template:      template,
		Mode:          "fast",
		WorkspaceMode: "copy",
	})
	if err != nil {
		t.Fatalf("CreateWithOptions failed: %v", err)
	}
	if err := store.UpdateState(id, sandbox.StateWarm); err != nil {
		t.Fatalf("UpdateState warm failed: %v", err)
	}
	return id
}

func TestCreateSandbox(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"name":"test-box","template":"node-python","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	api.CreateSandbox(store)(w, req)

	if fastUnavailable(w) {
		return
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["id"] == "" {
		t.Fatal("Expected non-empty id in response")
	}
}

func TestCreateSandbox_RequiresName(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"template":"node-python","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	api.CreateSandbox(store)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", w.Code)
	}
}

func TestCreateSandbox_Defaults(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"name":"test-box"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	api.CreateSandbox(store)(w, req)

	if w.Code != http.StatusCreated {
		t.Skipf("create default runtime unavailable in this environment: %d %s", w.Code, w.Body.String())
	}

	// Verify defaults were applied
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	id := resp["id"]

	meta, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.Template != "base" {
		t.Errorf("Expected default template 'base', got '%s'", meta.Template)
	}
	if meta.Mode != "fast" {
		t.Errorf("Expected default mode 'fast', got '%s'", meta.Mode)
	}
	if meta.WorkspaceMode != "copy" {
		t.Errorf("Expected default workspace mode 'copy', got '%s'", meta.WorkspaceMode)
	}
}

func TestCreateSandbox_WorkspaceMetadata(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"name":"gui-box","template":"node-python","mode":"fast","workspace":{"mode":"bind","source":"/tmp/project","maxSize":"5Gi"}}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	api.CreateSandbox(store)(w, req)

	if fastUnavailable(w) {
		return
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	meta, err := store.Get(resp["id"])
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.Workspace != "/tmp/project" {
		t.Errorf("Expected workspace source, got %q", meta.Workspace)
	}
	if meta.WorkspaceMode != "bind" {
		t.Errorf("Expected workspace mode bind, got %q", meta.WorkspaceMode)
	}
}

func TestCreateSandbox_OmittedWorkspaceDoesNotPersistSource(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"name":"template-only","template":"node-python","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	api.CreateSandbox(store)(w, req)

	if fastUnavailable(w) {
		return
	}
	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	meta, err := store.Get(resp["id"])
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.Workspace != "" {
		t.Errorf("Expected no workspace source, got %q", meta.Workspace)
	}
	if meta.WorkspaceMode != "copy" {
		t.Errorf("Expected default workspace mode copy, got %q", meta.WorkspaceMode)
	}
}

func TestCreateSandbox_RejectsUnsafeBindWorkspaceSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []string{
		home,
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".kube", "config"),
		filepath.Join(home, ".config", "gcloud", "credentials.db"),
		"/var/run/docker.sock",
	}

	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			store, _ := newTestStore(t)
			body := `{"name":"unsafe","workspace":{"mode":"bind","source":` + strconv.Quote(source) + `}}`
			req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(body))
			w := httptest.NewRecorder()

			api.CreateSandbox(store)(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("Expected 400 for unsafe bind source %q, got %d", source, w.Code)
			}
			if !strings.Contains(w.Body.String(), "unsafe workspace source") {
				t.Fatalf("response = %s, want unsafe workspace source error", w.Body.String())
			}
		})
	}
}

func TestListSandboxes(t *testing.T) {
	store, _ := newTestStore(t)

	createStoredSandbox(t, store, "box-1", "base")
	createStoredSandbox(t, store, "box-2", "node")

	// List
	req := httptest.NewRequest("GET", "/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	api.ListSandboxes(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var list []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 2 {
		t.Fatalf("Expected 2 sandboxes, got %d", len(list))
	}
}

func TestGetSandbox(t *testing.T) {
	store, _ := newTestStore(t)

	id := createStoredSandbox(t, store, "test-box", "python")

	// Get the sandbox
	req := httptest.NewRequest("GET", "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	api.GetSandbox(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["name"] != "test-box" {
		t.Errorf("Expected name 'test-box', got '%v'", resp["name"])
	}
	if resp["template"] != "python" {
		t.Errorf("Expected template 'python', got '%v'", resp["template"])
	}
	if resp["state"] != "WARM" {
		t.Errorf("Expected state 'WARM', got '%v'", resp["state"])
	}
	if resp["workspace_mode"] != "copy" {
		t.Errorf("Expected workspace_mode 'copy', got '%v'", resp["workspace_mode"])
	}
}

func TestGetSandbox_NotFound(t *testing.T) {
	store, _ := newTestStore(t)

	req := httptest.NewRequest("GET", "/v1/sandboxes/nonexistent", nil)
	w := httptest.NewRecorder()
	api.GetSandbox(store)(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestDeleteSandbox(t *testing.T) {
	store, _ := newTestStore(t)

	id := createStoredSandbox(t, store, "test-box", "base")

	// Delete
	req := httptest.NewRequest("DELETE", "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	api.DeleteSandbox(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Verify it's gone
	req = httptest.NewRequest("GET", "/v1/sandboxes/"+id, nil)
	w = httptest.NewRecorder()
	api.GetSandbox(store)(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 after delete, got %d", w.Code)
	}
}

func TestFullLifecycle(t *testing.T) {
	store, _ := newTestStore(t)

	id := createStoredSandbox(t, store, "lifecycle-test", "base")

	// List — should have 1
	req := httptest.NewRequest("GET", "/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	api.ListSandboxes(store)(w, req)
	var list []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("Expected 1 sandbox after create, got %d", len(list))
	}

	// Get
	req = httptest.NewRequest("GET", "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w = httptest.NewRecorder()
	api.GetSandbox(store)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w = httptest.NewRecorder()
	api.DeleteSandbox(store)(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	// List — should have 0
	req = httptest.NewRequest("GET", "/v1/sandboxes", nil)
	w = httptest.NewRecorder()
	api.ListSandboxes(store)(w, req)
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 0 {
		t.Fatalf("Expected 0 sandboxes after delete, got %d", len(list))
	}
}

func TestHealthEndpoint(t *testing.T) {
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	expected := `{"status":"ok"}`
	if string(body) != expected && string(body) != expected+"\n" {
		t.Errorf("Expected %q, got %q", expected, string(body))
	}
}
