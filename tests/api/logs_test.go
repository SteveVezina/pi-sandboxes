package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/pi-sandbox/pi/pkg/daemon"
)

// TestLogsAndHistory verifies F10: exec produces a log entry retrievable
// via /logs (full entries) and /logs/history (summary), with stdout and
// stderr content available through the action=stdout/stderr routes.
func TestLogsAndHistory(t *testing.T) {
	// logs.NewManager resolves its directory from $HOME, independent of
	// the test's isolated sandbox store; keep it off the real user home.
	t.Setenv("HOME", t.TempDir())

	store, _ := newTestStore(t)
	id := makeWarmExecSandbox(t, store, "logs-test")
	router := daemon.NewRouter(store)

	execBody := `{"command":"echo hello-stdout; echo hello-stderr 1>&2","timeoutMs":5000}`
	execReq := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewBufferString(execBody))
	execW := httptest.NewRecorder()
	router.ServeHTTP(execW, execReq)
	if execW.Code != http.StatusOK {
		t.Fatalf("exec: expected 200, got %d: %s", execW.Code, execW.Body.String())
	}

	// /logs — full entries
	logsReq := httptest.NewRequest("GET", "/v1/sandboxes/"+id+"/logs", nil)
	logsW := httptest.NewRecorder()
	router.ServeHTTP(logsW, logsReq)
	if logsW.Code != http.StatusOK {
		t.Fatalf("logs: expected 200, got %d: %s", logsW.Code, logsW.Body.String())
	}
	var logsResp struct {
		Entries []struct {
			Sequence int    `json:"sequence"`
			Command  string `json:"command"`
			ExitCode int    `json:"exitCode"`
		} `json:"entries"`
	}
	json.Unmarshal(logsW.Body.Bytes(), &logsResp)
	if len(logsResp.Entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logsResp.Entries))
	}
	if logsResp.Entries[0].ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", logsResp.Entries[0].ExitCode)
	}
	seq := logsResp.Entries[0].Sequence

	// /logs/history — summary
	historyReq := httptest.NewRequest("GET", "/v1/sandboxes/"+id+"/logs/history", nil)
	historyW := httptest.NewRecorder()
	router.ServeHTTP(historyW, historyReq)
	if historyW.Code != http.StatusOK {
		t.Fatalf("history: expected 200, got %d: %s", historyW.Code, historyW.Body.String())
	}
	var historyResp struct {
		Entries []struct {
			Sequence int `json:"sequence"`
		} `json:"entries"`
	}
	json.Unmarshal(historyW.Body.Bytes(), &historyResp)
	if len(historyResp.Entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(historyResp.Entries))
	}

	// stdout content
	stdoutReq := httptest.NewRequest("GET", fmtLogsURL(id, "stdout", seq), nil)
	stdoutW := httptest.NewRecorder()
	router.ServeHTTP(stdoutW, stdoutReq)
	if stdoutW.Code != http.StatusOK {
		t.Fatalf("stdout: expected 200, got %d: %s", stdoutW.Code, stdoutW.Body.String())
	}
	if got := stdoutW.Body.String(); got != "hello-stdout\n" {
		t.Errorf("stdout = %q, want %q", got, "hello-stdout\n")
	}

	// stderr content
	stderrReq := httptest.NewRequest("GET", fmtLogsURL(id, "stderr", seq), nil)
	stderrW := httptest.NewRecorder()
	router.ServeHTTP(stderrW, stderrReq)
	if stderrW.Code != http.StatusOK {
		t.Fatalf("stderr: expected 200, got %d: %s", stderrW.Code, stderrW.Body.String())
	}
	if got := stderrW.Body.String(); got != "hello-stderr\n" {
		t.Errorf("stderr = %q, want %q", got, "hello-stderr\n")
	}
}

func fmtLogsURL(id, action string, seq int) string {
	return "/v1/sandboxes/" + id + "/logs?action=" + action + "&sequence=" + strconv.Itoa(seq)
}
