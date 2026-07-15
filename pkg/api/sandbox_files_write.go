package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// maxInlineWriteBytes caps files/write payloads; the content travels
// through the container exec argv, which has platform limits.
const maxInlineWriteBytes = 1 << 20 // 1 MiB

// FilesWriteRequest is the request body for writing a file.
type FilesWriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FilesWriteSandbox returns an HTTP handler that writes a file into the
// sandbox workspace via the sandbox container.
func FilesWriteSandbox(store *sandbox.Store) http.HandlerFunc {
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

		abs, err := resolveWorkspacePath(req.Path)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if len(req.Content) > maxInlineWriteBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": fmt.Sprintf("content exceeds %d bytes", maxInlineWriteBytes)})
			return
		}

		// Write via exec inside the container so the file is owned by the
		// sandbox user (docker cp would leave it root-owned).
		encoded := base64.StdEncoding.EncodeToString([]byte(req.Content))
		script := "mkdir -p " + shellQuote(filepath.Dir(abs)) +
			" && printf '%s' " + shellQuote(encoded) + " | base64 -d > " + shellQuote(abs)
		if _, err := workspaceExec(r.Context(), id, script); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write file: " + err.Error()})
			return
		}

		store.UpdateLastUsed(id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":    id,
			"path":  abs,
			"bytes": len(req.Content),
		})
	}
}
