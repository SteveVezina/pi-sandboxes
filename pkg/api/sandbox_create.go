package api

import (
	"encoding/json"
	"net/http"

	"github.com/pi-sandbox/pi/pkg/session"
)

// CreateRequest is the request body for sandbox creation.
type CreateRequest struct {
	Template  string `json:"template"`
	Mode      string `json:"mode"`
	Name      string `json:"name"`
	TTL       int    `json:"ttlSeconds"`
	Workspace struct {
		Mode    string `json:"mode"`
		Source  string `json:"source"`
		MaxSize string `json:"maxSize"`
	} `json:"workspace"`
}

// CreateSandbox returns an HTTP handler that creates a sandbox session.
func CreateSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}
		if req.Template == "" {
			req.Template = "base"
		}
		if req.Mode == "" {
			req.Mode = "fast"
		}

		id, err := store.CreateWithOptions(session.CreateOptions{
			Name:          req.Name,
			Template:      req.Template,
			Mode:          req.Mode,
			TTL:           req.TTL,
			Workspace:     req.Workspace.Source,
			WorkspaceMode: req.Workspace.Mode,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Update state to warm
		store.UpdateState(id, session.StateWarm)

		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}
