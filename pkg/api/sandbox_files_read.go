package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// FilesReadRequest is the query parameter for file read.
type FilesReadRequest struct {
	Path string `json:"path"`
}

// FilesReadSandbox returns an HTTP handler that reads a file from the
// sandbox workspace via the sandbox container.
func FilesReadSandbox(store *sandbox.Store) http.HandlerFunc {
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

		abs, err := resolveWorkspacePath(path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		content, err := workspaceExec(r.Context(), id, "cat "+shellQuote(abs))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "read file: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"path":    abs,
			"content": content,
			"bytes":   len(content),
		})
	}
}

// FilesListSandbox returns an HTTP handler that lists files in the
// sandbox workspace via the sandbox container.
func FilesListSandbox(store *sandbox.Store) http.HandlerFunc {
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

		abs, err := resolveWorkspacePath(r.URL.Query().Get("path"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		out, err := workspaceExec(r.Context(), id, "ls -1Ap "+shellQuote(abs))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "list files: " + err.Error()})
			return
		}

		entries := []string{}
		for _, line := range strings.Split(out, "\n") {
			if line != "" {
				entries = append(entries, line)
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"path":    abs,
			"entries": entries,
			"count":   len(entries),
		})
	}
}
