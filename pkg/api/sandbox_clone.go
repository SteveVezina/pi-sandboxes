package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// CloneRequest is the request body for cloning a repository.
type CloneRequest struct {
	URL string `json:"url"`
}

// CloneSandbox returns an HTTP handler that clones a repository into the
// sandbox workspace, running git inside the sandbox container so the
// checkout lands on the daemon-managed workspace volume.
func CloneSandbox(store *sandbox.Store) http.HandlerFunc {
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

		// Parse request
		var req CloneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		req.URL = strings.TrimSpace(req.URL)
		if req.URL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		script := gitPreamble +
			"cd /workspace && if [ -n \"$(ls -A . 2>/dev/null)\" ]; then " +
			"echo 'workspace is not empty' >&2; exit 1; fi; " +
			"git clone " + shellQuote(req.URL) + " ."
		if _, err := workspaceExec(ctx, id, script); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clone failed: " + err.Error()})
			return
		}

		store.UpdateLastUsed(id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          id,
			"name":        meta.Name,
			"workspace":   workspaceRoot,
			"cloned_from": req.URL,
		})
	}
}
