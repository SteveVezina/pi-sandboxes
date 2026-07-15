package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestExecSandbox(t *testing.T) {
	store, _ := newTestStore(t)

	id := makeWarmExecSandbox(t, store, "exec-test")

	// Execute via router
	execBody := `{"command":"echo hello world","timeoutMs":5000}`
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result api.ExecResponse
	json.NewDecoder(w.Body).Decode(&result)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	if result.TimedOut {
		t.Error("Expected timedOut to be false")
	}
	if result.DurationMs <= 0 {
		t.Error("Expected positive duration")
	}
}

func TestExecSandbox_AcceptsValidNetworkMode(t *testing.T) {
	store, _ := newTestStore(t)

	id := makeWarmExecSandbox(t, store, "exec-network")

	execBody := `{"command":"echo network ok","timeoutMs":5000,"network":"restricted"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecSandbox_RejectsInvalidNetworkMode(t *testing.T) {
	store, _ := newTestStore(t)

	id := makeWarmExecSandbox(t, store, "exec-network")

	execBody := `{"command":"echo no","network":"wide-open"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid network mode") {
		t.Fatalf("response = %s, want invalid network mode error", w.Body.String())
	}
}

func TestExecSandbox_RejectsNonWarmState(t *testing.T) {
	store, _ := newTestStore(t)

	id, err := store.CreateWithOptions(session.CreateOptions{
		Name:     "busy-session",
		Template: "base",
		Mode:     "fast",
	})
	if err != nil {
		t.Fatalf("CreateWithOptions failed: %v", err)
	}
	if err := store.UpdateState(id, session.StateWarm); err != nil {
		t.Fatalf("UpdateState warm failed: %v", err)
	}
	if err := store.UpdateState(id, session.StateExecuting); err != nil {
		t.Fatalf("UpdateState executing failed: %v", err)
	}

	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(`{"command":"echo no"}`))
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()

	api.ExecSandbox(store)(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "WARM") {
		t.Fatalf("response = %s, want required WARM state", w.Body.String())
	}
}

func TestExecSandbox_NotFound(t *testing.T) {
	store, _ := newTestStore(t)

	req := httptest.NewRequest("POST", "/v1/sandboxes/nonexistent/exec",
		bytes.NewBufferString(`{"command":"echo test"}`))
	w := httptest.NewRecorder()
	api.ExecSandbox(store)(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestExecSandbox_BadRequest(t *testing.T) {
	store, _ := newTestStore(t)

	id := makeWarmExecSandbox(t, store, "test")

	// Bad exec request (invalid JSON)
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(`{invalid`))
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w := httptest.NewRecorder()
	api.ExecSandbox(store)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", w.Code)
	}
}

func makeWarmExecSandbox(t *testing.T, store *session.Store, name string) string {
	t.Helper()
	id, err := store.CreateWithOptions(session.CreateOptions{
		Name:     name,
		Template: "base",
		Mode:     "fast",
	})
	if err != nil {
		t.Fatalf("CreateWithOptions failed: %v", err)
	}
	if err := store.UpdateState(id, session.StateWarm); err != nil {
		t.Fatalf("UpdateState warm failed: %v", err)
	}
	return id
}

func TestExecRouter(t *testing.T) {
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	// Verify exec endpoint is registered
	req := httptest.NewRequest("POST", "/v1/sandboxes/test-id/exec", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should get 404 (sandbox not found), not 404 (route not found)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404 (sandbox not found), got %d", w.Code)
	}
}
