package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// PatchSandbox returns an HTTP handler that exports the workspace diff as
// a git patch computed inside the sandbox container.
func PatchSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if err := requireCompat(meta); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		start := time.Now()
		patch, err := workspaceDiff(ctx, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "patch failed: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          id,
			"name":        meta.Name,
			"patch":       patch,
			"timed_out":   false,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	}
}
