package api_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/session"
)

// streamRouter builds a minimal mux that mounts the exec handler.
func streamRouter(store *session.Store) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/v1/sandboxes/{id}/exec", api.ExecSandbox(store)).Methods("POST")
	return r
}

// makeStreamSandbox creates a sandbox in the store for test use.
func makeStreamSandbox(t *testing.T, store *session.Store) string {
	t.Helper()
	id, err := store.Create("test-sb", "node-python", "fast")
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	if err := store.UpdateState(id, session.StateWarm); err != nil {
		t.Fatalf("warm sandbox: %v", err)
	}
	return id
}

// TestExecHandler_NonStreaming verifies the backwards-compatible JSON response.
func TestExecHandler_NonStreaming(t *testing.T) {
	store, _ := newTestStore(t)
	id := makeStreamSandbox(t, store)
	router := streamRouter(store)

	body, _ := json.Marshal(map[string]interface{}{
		"command": "echo hello",
		"cwd":     "/tmp",
	})
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["exitCode"]; !ok {
		t.Error("missing exitCode in non-streaming response")
	}
	if _, ok := resp["stdout"]; !ok {
		t.Error("missing stdout in non-streaming response")
	}
}

// TestExecHandler_StreamingQueryParam verifies NDJSON streaming via ?stream=true (AC-7).
func TestExecHandler_StreamingQueryParam(t *testing.T) {
	store, _ := newTestStore(t)
	id := makeStreamSandbox(t, store)
	router := streamRouter(store)

	body, _ := json.Marshal(map[string]interface{}{
		"command": "echo streamed",
		"cwd":     "/tmp",
	})
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec?stream=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/x-ndjson" {
		t.Errorf("expected Content-Type application/x-ndjson, got %q", ct)
	}

	events := parseStreamEvents(t, w.Body.String())
	assertStreamType(t, events, "stdout")
	assertStreamType(t, events, "done")
}

// TestExecHandler_StreamingAcceptHeader verifies NDJSON streaming via
// Accept: application/x-ndjson (AC-7).
func TestExecHandler_StreamingAcceptHeader(t *testing.T) {
	store, _ := newTestStore(t)
	id := makeStreamSandbox(t, store)
	router := streamRouter(store)

	body, _ := json.Marshal(map[string]interface{}{
		"command": "printf 'a\\nb\\nc\\n'",
		"cwd":     "/tmp",
	})
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	events := parseStreamEvents(t, w.Body.String())
	assertStreamType(t, events, "done")

	for _, ev := range events {
		if ev["type"] == "done" {
			if _, ok := ev["exitCode"]; !ok {
				t.Error("done event missing exitCode")
			}
			if _, ok := ev["durationMs"]; !ok {
				t.Error("done event missing durationMs")
			}
		}
	}
}

// TestExecHandler_StreamingNonZeroExit verifies done.exitCode reflects process exit.
func TestExecHandler_StreamingNonZeroExit(t *testing.T) {
	store, _ := newTestStore(t)
	id := makeStreamSandbox(t, store)
	router := streamRouter(store)

	body, _ := json.Marshal(map[string]interface{}{
		"command": "exit 7",
		"cwd":     "/tmp",
	})
	req := httptest.NewRequest("POST", "/v1/sandboxes/"+id+"/exec?stream=true", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	events := parseStreamEvents(t, w.Body.String())
	for _, ev := range events {
		if ev["type"] == "done" {
			code, _ := ev["exitCode"].(float64)
			if int(code) != 7 {
				t.Errorf("expected exitCode 7, got %v", ev["exitCode"])
			}
			return
		}
	}
	t.Error("no done event in streaming response")
}

func parseStreamEvents(t *testing.T, s string) []map[string]interface{} {
	t.Helper()
	var events []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev map[string]interface{}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("parse NDJSON line %q: %v", line, err)
		}
		events = append(events, ev)
	}
	return events
}

func assertStreamType(t *testing.T, events []map[string]interface{}, typ string) {
	t.Helper()
	for _, ev := range events {
		if ev["type"] == typ {
			return
		}
	}
	t.Errorf("no event of type %q found in %d events", typ, len(events))
}
