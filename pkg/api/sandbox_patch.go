package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/git"
	"github.com/pi-sandbox/pi/pkg/sandbox"
	"github.com/pi-sandbox/pi/pkg/workspace"
)

// PatchSandbox returns an HTTP handler that exports workspace as a git patch.
func PatchSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		// Create workspace manager
		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		// Initialize git repo if needed
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		if err := git.InitIfNotRepo(ctx, mgr.Dir()); err != nil {
			// Non-fatal — just log and continue
		}

		// Export patch
		patch, err := git.Patch(ctx, mgr.Dir())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "patch failed: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          id,
			"name":        meta.Name,
			"patch":       patch.Patch,
			"timed_out":   patch.TimedOut,
			"duration_ms": patch.DurationMs,
		})
	}
}
