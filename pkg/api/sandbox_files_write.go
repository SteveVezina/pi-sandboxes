package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/workspace"
)

// FilesWriteRequest is the request body for writing a file.
type FilesWriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FilesWriteSandbox returns an HTTP handler that writes a file to a sandbox workspace.
func FilesWriteSandbox(store *session.Store) http.HandlerFunc {
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
		var req FilesWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path is required"})
			return
		}
		if req.Content == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
			return
		}

		// Write the file
		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.WriteFile(req.Path, []byte(req.Content)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write file: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":     id,
			"path":   req.Path,
			"bytes":  len(req.Content),
		})
	}
}
