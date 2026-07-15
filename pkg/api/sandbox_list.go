package api

import (
	"net/http"
	"time"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// ListSandboxes returns an HTTP handler that lists sandbox sessions.
func ListSandboxes(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := store.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		type SandboxInfo struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Template      string `json:"template"`
			Mode          string `json:"mode"`
			State         string `json:"state"`
			Workspace     string `json:"workspace"`
			WorkspaceMode string `json:"workspace_mode"`
			CreatedAt     string `json:"created_at"`
			UpdatedAt     string `json:"updated_at"`
			LastUsed      string `json:"last_used"`
		}

		var list []SandboxInfo
		for _, id := range ids {
			meta, err := store.Get(id)
			if err != nil {
				continue
			}
			list = append(list, SandboxInfo{
				ID:            meta.ID,
				Name:          meta.Name,
				Template:      meta.Template,
				Mode:          meta.Mode,
				State:         string(meta.State),
				Workspace:     meta.Workspace,
				WorkspaceMode: meta.WorkspaceMode,
				CreatedAt:     meta.CreatedAt.Format(time.RFC3339Nano),
				UpdatedAt:     meta.UpdatedAt.Format(time.RFC3339Nano),
				LastUsed:      meta.LastUsedAt.Format(time.RFC3339Nano),
			})
		}

		if list == nil {
			list = []SandboxInfo{}
		}

		writeJSON(w, http.StatusOK, list)
	}
}
