package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/exec"
	"github.com/pi-sandbox/pi/pkg/logs"
	"github.com/pi-sandbox/pi/pkg/network"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// ExecRequest is the API request for command execution.
type ExecRequest struct {
	Command        string `json:"command"`
	Cwd            string `json:"cwd"`
	TimeoutMs      int64  `json:"timeoutMs"`
	MaxOutputBytes int64  `json:"maxOutputBytes"`
	Network        string `json:"network"`
}

// ExecResponse is the API response for command execution (non-streaming).
type ExecResponse struct {
	ExitCode   int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	Truncated  bool   `json:"truncated"`
	TimedOut   bool   `json:"timedOut"`
}

// wantsStream reports whether the request opts into streaming NDJSON output.
func wantsStream(r *http.Request) bool {
	if r.URL.Query().Get("stream") == "true" {
		return true
	}
	return r.Header.Get("Accept") == "application/x-ndjson"
}

// ExecSandbox returns an HTTP handler that executes a command in a sandbox.
func ExecSandbox(store *sandbox.Store) http.HandlerFunc {
	engine := exec.NewEngine(8 * 1024 * 1024) // 8 MiB default

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists and get its metadata
		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if !requireSandboxState(w, meta, sandbox.StateWarm) {
			return
		}

		// Parse request
		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Network != "" {
			policy := network.DefaultPolicy().ApplyNetworkMode(network.Mode(req.Network))
			if err := policy.Validate(); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		}

		execReq := &exec.Request{
			Command:     req.Command,
			Cwd:         req.Cwd,
			Timeout:     time.Duration(req.TimeoutMs) * time.Millisecond,
			MaxOutput:   req.MaxOutputBytes,
			NetworkMode: req.Network,
		}
		if execReq.Timeout == 0 {
			execReq.Timeout = 120 * time.Second
		}
		if execReq.MaxOutput == 0 {
			execReq.MaxOutput = 8 * 1024 * 1024
		}
		if execReq.Cwd == "" {
			execReq.Cwd = "/workspace"
		}

		if err := store.UpdateLastUsed(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := store.UpdateState(id, sandbox.StateExecuting); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer store.UpdateState(id, sandbox.StateWarm)

		ctx, cancel := contextWithTimeout(r.Context(), execReq.Timeout)
		defer cancel()

		// Check if we should use the compat backend
		if strings.EqualFold(meta.Mode, "compat") {
			if err := execInContainer(store, id, meta, ctx, execReq, w, wantsStream(r)); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		} else {
			// Use the standard exec engine (host execution)
			if wantsStream(r) {
				w.Header().Set("Content-Type", "application/x-ndjson")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("Cache-Control", "no-cache")
				w.WriteHeader(http.StatusOK)

				bw := bufio.NewWriter(w)
				_ = engine.RunStream(ctx, execReq, bw)
				_ = bw.Flush()
			} else {
				result, err := engine.Run(ctx, execReq)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
					return
				}
				_, _ = logs.NewManager(id).Record(execReq.Command, result.ExitCode,
					result.DurationMs, result.TimedOut, result.Truncated, result.Stdout, result.Stderr)
				writeJSON(w, http.StatusOK, ExecResponse{
					ExitCode:   result.ExitCode,
					DurationMs: result.DurationMs,
					Stdout:     result.Stdout,
					Stderr:     result.Stderr,
					Truncated:  result.Truncated,
					TimedOut:   result.TimedOut,
				})
			}
		}
	}
}

// execInContainer executes a command inside an OCI container using the compat backend.
func execInContainer(store *sandbox.Store, id string, meta *sandbox.Meta, ctx context.Context, execReq *exec.Request, w http.ResponseWriter, streaming bool) error {
	// Ensure OCI runtime is available
	if err := compat.EnsureRuntimeAvailable(); err != nil {
		return fmt.Errorf("compat backend: %w", err)
	}

	// Create container spec
	spec := &compat.ContainerSpec{
		ID:        id,
		Name:      "pi-sandbox-" + id[:8],
		Image:     "debian:bookworm-slim",
		Workspace: compatWorkspaceSource(id, meta),
		Artifacts: compatArtifactsSource(id),
		Network:   sandboxEgressNetwork(meta),
		Mode:      meta.Mode,
		Template:  meta.Template,
	}

	// Create the container (or get existing)
	container, err := compat.CreateContainer(spec)
	if err != nil {
		// If container already exists, try to use it
		exists, _ := compat.ContainerExists(spec.Name)
		if exists {
			container = &compat.Container{
				ID:    id,
				Spec:  spec,
				Ready: true,
			}
		} else {
			return fmt.Errorf("create container: %w", err)
		}
	}

	// Start the container if not already running
	if !container.Ready {
		if err := container.Start(); err != nil {
			return fmt.Errorf("start container: %w", err)
		}
	}

	// Execute the command
	result, err := container.Exec(ctx, execReq.Command)
	if err != nil {
		return fmt.Errorf("exec in container: %w", err)
	}

	// Record the run in the sandbox exec history (best-effort).
	_, _ = logs.NewManager(id).Record(execReq.Command, result.ExitCode,
		result.DurationMs, result.TimedOut, result.Truncated, result.Stdout, result.Stderr)

	if streaming {
		// Streaming path
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)

		// Write stdout events
		lines := strings.Split(result.Stdout, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			event := map[string]interface{}{
				"type": "stdout",
				"data": line,
			}
			data, _ := json.Marshal(event)
			w.Write(append(data, '\n'))
			if f, ok := w.(interface{ Flush() error }); ok {
				_ = f.Flush()
			}
		}

		// Write stderr events
		lines = strings.Split(result.Stderr, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			event := map[string]interface{}{
				"type": "stderr",
				"data": line,
			}
			data, _ := json.Marshal(event)
			w.Write(append(data, '\n'))
			if f, ok := w.(interface{ Flush() error }); ok {
				_ = f.Flush()
			}
		}

		// Write done event
		doneEvent := map[string]interface{}{
			"type":       "done",
			"exitCode":   result.ExitCode,
			"durationMs": result.DurationMs,
			"timedOut":   result.TimedOut,
		}
		data, _ := json.Marshal(doneEvent)
		w.Write(append(data, '\n'))
		if f, ok := w.(interface{ Flush() error }); ok {
			_ = f.Flush()
		}
	} else {
		// Non-streaming path
		writeJSON(w, http.StatusOK, ExecResponse{
			ExitCode:   result.ExitCode,
			DurationMs: result.DurationMs,
			Stdout:     result.Stdout,
			Stderr:     result.Stderr,
			TimedOut:   result.TimedOut,
		})
	}

	return nil
}

// contextWithTimeout creates a context with the given timeout.
func contextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
