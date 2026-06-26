package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/session"
)

func newTestStore(t *testing.T) (*session.Store, string) {
	tmpDir := filepath.Join(os.TempDir(), "pi-api-test")
	os.RemoveAll(tmpDir)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	store := session.NewStore(tmpDir)
	return store, tmpDir
}

func TestCreateSandbox(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"name":"test-box","template":"node-python","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	api.CreateSandbox(store)(w, req)

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
		t.Fatalf("Expected 201, got %d", w.Code)
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
}

func TestListSandboxes(t *testing.T) {
	store, _ := newTestStore(t)

	// Create two sandboxes
	reqBody1 := `{"name":"box-1","template":"base","mode":"fast"}`
	req1 := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody1))
	w1 := httptest.NewRecorder()
	api.CreateSandbox(store)(w1, req1)

	reqBody2 := `{"name":"box-2","template":"node","mode":"fast"}`
	req2 := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody2))
	w2 := httptest.NewRecorder()
	api.CreateSandbox(store)(w2, req2)

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

	// Create a sandbox
	reqBody := `{"name":"test-box","template":"python","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	// Get the sandbox
	req = httptest.NewRequest("GET", "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w = httptest.NewRecorder()
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

	// Create a sandbox
	reqBody := `{"name":"test-box","template":"base","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	// Delete
	req = httptest.NewRequest("DELETE", "/v1/sandboxes/"+id, nil)
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w = httptest.NewRecorder()
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

	// Create
	reqBody := `{"name":"lifecycle-test","template":"base","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	// List — should have 1
	req = httptest.NewRequest("GET", "/v1/sandboxes", nil)
	w = httptest.NewRecorder()
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
