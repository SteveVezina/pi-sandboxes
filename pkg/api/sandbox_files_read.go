package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/workspace"
)

// FilesReadRequest is the query parameter for file read.
type FilesReadRequest struct {
	Path string `json:"path"`
}

// FilesReadSandbox returns an HTTP handler that reads a file from a sandbox workspace.
func FilesReadSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		// Get path from query or body
		var path string
		if path = r.URL.Query().Get("path"); path == "" {
			var req FilesReadRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				path = req.Path
			}
		}
		if path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path is required (query ?path= or body)"})
			return
		}

		// Read the file
		mgr := workspace.NewManager(id, workspace.ModeCopy)
		data, err := mgr.ReadFile(path)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "read file: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"path":    path,
			"content": string(data),
			"bytes":   len(data),
		})
	}
}
