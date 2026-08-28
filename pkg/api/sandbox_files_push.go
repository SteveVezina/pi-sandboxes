package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// FilesPushRequest is the request body for pushing a host file/directory
// into the sandbox workspace.
type FilesPushRequest struct {
	Src  string `json:"src"`
	Dest string `json:"dest"`
}

// FilesPushSandbox returns an HTTP handler that copies a host path into
// the sandbox workspace (SPEC.md §12.6).
func FilesPushSandbox(store *sandbox.Store) http.HandlerFunc {
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

		var req FilesPushRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Src == "" || req.Dest == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "src and dest are required"})
			return
		}

		if _, err := os.Stat(req.Src); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("source not found: %v", err)})
			return
		}

		dest, err := resolveWorkspacePath(req.Dest)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		c, err := compatContainerHandle(id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		if err := c.CopyTo(req.Src, dest); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("push to %s: %v", dest, err)})
			return
		}

		// docker cp leaves the copied path root-owned; hand it back to the
		// sandbox user (SPEC.md §8 — sandbox processes run unprivileged).
		if err := c.ExecAsRoot(r.Context(), "chown -R 1000:1000 "+shellQuote(dest)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("chown %s: %v", dest, err)})
			return
		}

		store.UpdateLastUsed(id)

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":   id,
			"src":  req.Src,
			"dest": dest,
		})
	}
}
