package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
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
		var req AgentRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		sandboxID := r.PathValue("id")
		if sandboxID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sandbox ID is required"})
			return
		}

		if req.Agent == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "agent name is required"})
			return
		}

		// Verify sandbox exists and is WARM
		meta, err := store.Get(sandboxID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if meta.State != sandbox.StateWarm {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("sandbox is not WARM (state: %s)", meta.State)})
			return
		}

		// Create the agent run
		runID := uuid.New().String()
		run, err := runStore.Create(runID, sandboxID, req.Agent, req.RepoURL, req.Prompt)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}

		// Transition to STARTING (actual start happens asynchronously)
		runStore.UpdateState(runID, sandbox.RunStateStarting, 0, "")

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"run_id":    run.RunID,
			"sandbox_id": run.SandboxID,
			"agent":     run.AgentName,
			"state":     run.State,
		})
	}
}

// GetAgentRun returns an HTTP handler that inspects an agent run.
func GetAgentRun(runStore *sandbox.AgentRunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runID := r.PathValue("id")
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
		runID := r.PathValue("id")
		if runID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run ID is required"})
			return
		}

		run, err := runStore.Get(runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}

		if run.State != sandbox.RunStateRunning && run.State != sandbox.RunStateStarting {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("cannot cancel run in state %s", run.State)})
			return
		}

		runStore.UpdateState(runID, sandbox.RunStateCancelled, 0, "")

		writeJSON(w, http.StatusOK, map[string]string{"run_id": runID, "state": string(sandbox.RunStateCancelled)})
	}
}

// ListAgentRuns returns an HTTP handler that lists all agent runs.
func ListAgentRuns(runStore *sandbox.AgentRunStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runs := runStore.List()
		writeJSON(w, http.StatusOK, runs)
	}
}

// CompleteAgentRun is a helper for the daemon to signal run completion.
func CompleteAgentRun(runStore *sandbox.AgentRunStore, runID string, exitCode int, errMsg string) error {
	_, err := runStore.UpdateState(runID, sandbox.RunStateCompleted, exitCode, errMsg)
	return err
}

// FailAgentRun is a helper for the daemon to signal run failure.
func FailAgentRun(runStore *sandbox.AgentRunStore, runID string, errMsg string) error {
	_, err := runStore.UpdateState(runID, sandbox.RunStateFailed, 1, errMsg)
	return err
}

// StartAgentRunAsync is a helper for the daemon to start an agent run asynchronously.
func StartAgentRunAsync(runStore *sandbox.AgentRunStore, runID string) {
	// Transition to RUNNING after a short delay (actual start logic in daemon)
	time.Sleep(100 * time.Millisecond)
	runStore.UpdateState(runID, sandbox.RunStateRunning, 0, "")
}
