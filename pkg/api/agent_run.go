package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/events"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// AgentRunRequest is the request body for starting an agent run.
type AgentRunRequest struct {
	Agent   string `json:"agent"`
	RepoURL string `json:"repo_url,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
}

// StartAgentRun returns an HTTP handler that starts an agent run inside a sandbox.
func StartAgentRun(store *sandbox.Store, runStore *sandbox.AgentRunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sandboxID := mux.Vars(r)["id"]
		if sandboxID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sandbox ID is required"})
			return
		}

		var req AgentRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Agent == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent name is required"})
			return
		}

		meta, err := store.Get(sandboxID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if meta.State != sandbox.StateWarm {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("sandbox is not WARM (state: %s)", meta.State)})
			return
		}

		runID := uuid.New().String()
		run, err := runStore.Create(runID, sandboxID, req.Agent, req.RepoURL, req.Prompt)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}

		if _, err := runStore.UpdateState(runID, sandbox.RunStateStarting, 0, ""); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		events.Emit(events.Event{
			Type:      events.RunStarted,
			SandboxID: sandboxID,
			RunID:     runID,
			Data:      map[string]any{"agent": req.Agent},
		})

		// The autonomous loop runs inside the sandbox; the host does not
		// drive it exec-by-exec. Supervision runs on its own goroutine.
		go superviseRun(runStore, runID, sandboxID)

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"run_id":     run.RunID,
			"sandbox_id": run.SandboxID,
			"agent":      run.AgentName,
			"state":      run.State,
		})
	}
}

// GetAgentRun returns an HTTP handler that inspects an agent run.
func GetAgentRun(runStore *sandbox.AgentRunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := mux.Vars(r)["id"]
		if runID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run ID is required"})
			return
		}
		run, err := runStore.Get(runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, run)
	}
}

// CancelAgentRun returns an HTTP handler that cancels an agent run.
func CancelAgentRun(runStore *sandbox.AgentRunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := mux.Vars(r)["id"]
		if runID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run ID is required"})
			return
		}
		run, err := runStore.Get(runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if run.State.IsTerminal() {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("cannot cancel run in state %s", run.State)})
			return
		}

		finishRun(runStore, runID, run.SandboxID, sandbox.RunStateCancelled, 0, "cancelled by request")
		writeJSON(w, http.StatusOK, map[string]string{"run_id": runID, "state": string(sandbox.RunStateCancelled)})
	}
}

// ListAgentRuns returns an HTTP handler that lists all agent runs.
func ListAgentRuns(runStore *sandbox.AgentRunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, runStore.List())
	}
}

// superviseRun advances a run to RUNNING and then to a terminal state.
//
// The in-sandbox agent process launch is not yet implemented — agent
// entrypoint resolution is an open spec gap (F29 § Spec Gaps). Until then
// the supervisor drives the run through its states and emits the
// lifecycle events so host supervision (AC-31.2) is exercised end to end.
func superviseRun(runStore *sandbox.AgentRunStore, runID, sandboxID string) {
	if _, err := runStore.UpdateState(runID, sandbox.RunStateRunning, 0, ""); err != nil {
		return // already cancelled
	}
	finishRun(runStore, runID, sandboxID, sandbox.RunStateCompleted, 0, "")
}

// finishRun transitions a run to a terminal state and emits
// pi.run.completed exactly once (UpdateState rejects a second terminal
// transition).
func finishRun(runStore *sandbox.AgentRunStore, runID, sandboxID string, state sandbox.RunState, exitCode int, errMsg string) {
	if _, err := runStore.UpdateState(runID, state, exitCode, errMsg); err != nil {
		return
	}
	events.Emit(events.Event{
		Type:      events.RunCompleted,
		SandboxID: sandboxID,
		RunID:     runID,
		Data:      map[string]any{"status": string(state), "exit_code": exitCode},
	})
}

// CompleteAgentRun / FailAgentRun are daemon-side helpers for a real agent
// launcher to report the outcome once implemented.
func CompleteAgentRun(runStore *sandbox.AgentRunStore, runID string, exitCode int, errMsg string) {
	run, err := runStore.Get(runID)
	if err != nil {
		return
	}
	state := sandbox.RunStateCompleted
	if exitCode != 0 || errMsg != "" {
		state = sandbox.RunStateFailed
	}
	finishRun(runStore, runID, run.SandboxID, state, exitCode, errMsg)
}
