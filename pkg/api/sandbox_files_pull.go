package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// FilesPullRequest is the request body for pulling a file/directory from
// the sandbox workspace to a host destination.
type FilesPullRequest struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

// FilesPullSandbox returns an HTTP handler that copies a workspace path
// out of the sandbox to a host destination (SPEC.md §12.6). This is a
// debug/inspection copy, distinct from the deliverable output channel
// (`POST /v1/sandboxes/{id}/output`).
func FilesPullSandbox(store *sandbox.Store) http.HandlerFunc {
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

		var req FilesPullRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Src == "" || req.Dest == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "src and dest are required"})
			return
		}

		src, err := resolveWorkspacePath(req.Src)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if err := os.MkdirAll(filepath.Dir(req.Dest), 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("prepare destination: %v", err)})
			return
		}

		c, err := compatContainerHandle(id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		if err := c.CopyFrom(src, req.Dest); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("pull %s: %v", src, err)})
			return
		}

		store.UpdateLastUsed(id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":   id,
			"src":  src,
			"dest": req.Dest,
		})
	}
}
