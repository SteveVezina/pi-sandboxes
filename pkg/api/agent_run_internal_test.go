package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/events"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func agentRunFixture(t *testing.T, state sandbox.State) (*sandbox.Store, *sandbox.AgentRunStore, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sandboxes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := sandbox.NewStore(dir)
	id, err := store.CreateWithOptions(sandbox.CreateOptions{Name: "a", Template: "base", Mode: "fast"})
	if err != nil {
		t.Fatal(err)
	}
	if state != "" {
		store.UpdateState(id, state)
	}
	return store, sandbox.NewAgentRunStore(), id
}

func startRun(t *testing.T, store *sandbox.Store, rs *sandbox.AgentRunStore, sandboxID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/"+sandboxID+"/agent-run", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": sandboxID})
	w := httptest.NewRecorder()
	StartAgentRun(store, rs)(w, req)
	return w
}

func drainEvents(t *testing.T, ch chan events.Event, want map[string]int, timeout time.Duration) map[string]int {
	t.Helper()
	got := map[string]int{}
	deadline := time.After(timeout)
	total := 0
	for _, n := range want {
		total += n
	}
	for i := 0; i < total; i++ {
		select {
		case e := <-ch:
			got[e.Type]++
		case <-deadline:
			return got
		}
	}
	return got
}

func TestStartAgentRun_WarmSandbox_EmitsStartedThenCompleted(t *testing.T) {
	ch := captureEvents(t)
	store, rs, id := agentRunFixture(t, sandbox.StateWarm)

	w := startRun(t, store, rs, id, `{"agent":"demo"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body)
	}

	got := drainEvents(t, ch, map[string]int{events.RunStarted: 1, events.RunCompleted: 1}, 2*time.Second)
	if got[events.RunStarted] != 1 || got[events.RunCompleted] != 1 {
		t.Fatalf("events = %v, want one started + one completed", got)
	}
}

func TestStartAgentRun_NotWarm_409(t *testing.T) {
	captureEvents(t)
	store, rs, id := agentRunFixture(t, sandbox.StateCreating)
	if w := startRun(t, store, rs, id, `{"agent":"demo"}`); w.Code != http.StatusConflict {
		t.Fatalf("want 409, got %d", w.Code)
	}
}

func TestStartAgentRun_MissingAgent_400(t *testing.T) {
	captureEvents(t)
	store, rs, id := agentRunFixture(t, sandbox.StateWarm)
	if w := startRun(t, store, rs, id, `{}`); w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCancelAgentRun_EmitsCompletedOnce_ThenConflict(t *testing.T) {
	ch := captureEvents(t)
	store, rs, id := agentRunFixture(t, sandbox.StateWarm)

	// Pre-create a run stuck in STARTING so supervise doesn't race us.
	run, err := rs.Create("run-1", id, "demo", "", "")
	if err != nil {
		t.Fatal(err)
	}
	rs.UpdateState(run.RunID, sandbox.RunStateStarting, 0, "")

	cancel := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/agent-runs/run-1/cancel", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "run-1"})
		w := httptest.NewRecorder()
		CancelAgentRun(rs)(w, req)
		return w
	}

	if w := cancel(); w.Code != http.StatusOK {
		t.Fatalf("first cancel: want 200, got %d: %s", w.Code, w.Body)
	}
	if w := cancel(); w.Code != http.StatusConflict {
		t.Fatalf("second cancel: want 409, got %d", w.Code)
	}

	got := drainEvents(t, ch, map[string]int{events.RunCompleted: 1}, time.Second)
	if got[events.RunCompleted] != 1 {
		t.Fatalf("want exactly one pi.run.completed, got %v", got)
	}
	_ = store
}
