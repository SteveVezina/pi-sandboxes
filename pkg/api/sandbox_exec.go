package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/exec"
	"github.com/pi-sandbox/pi/pkg/session"
)

// ExecRequest is the API request for command execution.
type ExecRequest struct {
	Command       string        `json:"command"`
	Cwd           string        `json:"cwd"`
	TimeoutMs     int64         `json:"timeoutMs"`
	MaxOutputBytes int64        `json:"maxOutputBytes"`
	Network       string        `json:"network"`
}

// ExecResponse is the API response for command execution.
type ExecResponse struct {
	ExitCode    int    `json:"exitCode"`
	DurationMs  int64  `json:"durationMs"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	Truncated   bool   `json:"truncated"`
	TimedOut    bool   `json:"timedOut"`
}

// ExecSandbox returns an HTTP handler that executes a command in a sandbox.
func ExecSandbox(store *session.Store) http.HandlerFunc {
	engine := exec.NewEngine(8 * 1024 * 1024) // 8 MiB default

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		// Parse request
		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		// Build exec request
		execReq := &exec.Request{
			Command:     req.Command,
			Cwd:         req.Cwd,
			Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
			MaxOutput:   req.MaxOutputBytes,
		}
		if execReq.Timeout == 0 {
			execReq.Timeout = 120 * time.Second
		}
		if execReq.MaxOutput == 0 {
			execReq.MaxOutput = 8 * 1024 * 1024
		}
		if execReq.Cwd == "" {
			execReq.Cwd = "/tmp"
		}

		// Update last used time
		store.UpdateLastUsed(id)

		// Transition to executing
		store.UpdateState(id, session.StateExecuting)

		// Execute
		ctx, cancel := contextWithTimeout(r.Context(), execReq.Timeout)
		defer cancel()

		result, err := engine.Run(ctx, execReq)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Transition back to warm
		store.UpdateState(id, session.StateWarm)

		// Write response
		writeJSON(w, http.StatusOK, ExecResponse{
			ExitCode:    result.ExitCode,
			DurationMs:  result.DurationMs,
			Stdout:      result.Stdout,
			Stderr:      result.Stderr,
			Truncated:   result.Truncated,
			TimedOut:    result.TimedOut,
		})
	}
}

// contextWithTimeout creates a context with the given timeout.
func contextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
