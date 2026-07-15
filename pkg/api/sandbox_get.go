package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// GetSandbox returns an HTTP handler that gets a sandbox by ID.
func GetSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":             meta.ID,
			"name":           meta.Name,
			"template":       meta.Template,
			"mode":           meta.Mode,
			"state":          string(meta.State),
			"created_at":     meta.CreatedAt,
			"updated_at":     meta.UpdatedAt,
			"ttl_seconds":    meta.TTL,
			"last_used":      meta.LastUsedAt,
			"workspace":      meta.Workspace,
			"workspace_mode": meta.WorkspaceMode,
		})
	}
}
