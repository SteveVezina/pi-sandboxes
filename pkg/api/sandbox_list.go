package api

import (
	"net/http"

	"github.com/pi-sandbox/pi/pkg/session"
)

// ListSandboxes returns an HTTP handler that lists sandbox sessions.
func ListSandboxes(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := store.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		type SandboxInfo struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			State string `json:"state"`
		}

		var list []SandboxInfo
		for _, id := range ids {
			meta, err := store.Get(id)
			if err != nil {
				continue
			}
			list = append(list, SandboxInfo{
				ID:    meta.ID,
				Name:  meta.Name,
				State: string(meta.State),
			})
		}

		writeJSON(w, http.StatusOK, list)
	}
}
