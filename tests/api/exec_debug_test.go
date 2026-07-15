package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/daemon"
)

func TestExecSandboxDebug(t *testing.T) {
	store, _ := newTestStore(t)
	id := makeWarmExecSandbox(t, store, "exec-test")
	t.Logf("Sandbox ID: %s", id)

	// Execute via router - use /tmp as working directory
	execBody := `{"command":"echo hello world","cwd":"/tmp","timeoutMs":5000}`
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	router := daemon.NewRouter(store)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Logf("Exec response: %d %s", w.Code, w.Body.String())

	// Check the response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var result api.ExecResponse
	json.NewDecoder(w.Body).Decode(&result)

	t.Logf("Exit code: %d", result.ExitCode)
	t.Logf("Stdout: %s", result.Stdout)
	t.Logf("Stderr: %s", result.Stderr)
	t.Logf("Duration: %d ms", result.DurationMs)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	if result.DurationMs <= 0 {
		t.Error("Expected positive duration")
	}
}
