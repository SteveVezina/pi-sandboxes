package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/git"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/workspace"
)

// CloneRequest is the request body for cloning a repository.
type CloneRequest struct {
	URL string `json:"url"`
}

// CloneSandbox returns an HTTP handler that clones a repository into a sandbox workspace.
func CloneSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		// Parse request
		var req CloneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.URL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
			return
		}

		// Create workspace manager
		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create workspace: " + err.Error()})
			return
		}

		// Clone the repository
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		if _, err := git.Clone(ctx, req.URL, mgr.Dir()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clone failed: " + err.Error()})
			return
		}

		// Update last used
		store.UpdateLastUsed(id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          id,
			"name":        meta.Name,
			"workspace":   mgr.Dir(),
			"cloned_from": req.URL,
		})
	}
}
