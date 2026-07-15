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
)

func TestExecSandbox(t *testing.T) {
	store, _ := newTestStore(t)

	// Create a sandbox
	reqBody := `{"name":"exec-test","template":"base","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	// Execute via router
	execBody := `{"command":"echo hello world","timeoutMs":5000}`
	req = httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w = httptest.NewRecorder()
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

	reqBody := `{"name":"exec-network","template":"base","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	execBody := `{"command":"echo network ok","timeoutMs":5000,"network":"restricted"}`
	req = httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExecSandbox_RejectsInvalidNetworkMode(t *testing.T) {
	store, _ := newTestStore(t)

	reqBody := `{"name":"exec-network","template":"base","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	execBody := `{"command":"echo no","network":"wide-open"}`
	req = httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid network mode") {
		t.Fatalf("response = %s, want invalid network mode error", w.Body.String())
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

	// Create a sandbox first
	reqBody := `{"name":"test","template":"base","mode":"fast"}`
	req := httptest.NewRequest("POST", "/v1/sandboxes", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	api.CreateSandbox(store)(w, req)

	var createResp map[string]string
	json.NewDecoder(w.Body).Decode(&createResp)
	id := createResp["id"]

	// Bad exec request (invalid JSON)
	req = httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(`{invalid`))
	req = mux.SetURLVars(req, map[string]string{"id": id})
	w = httptest.NewRecorder()
	api.ExecSandbox(store)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", w.Code)
	}
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
