package api

import (
	"encoding/json"
	"net/http"

	pictx "github.com/pi-sandbox/pi/pkg/context"
)

type contextInfo struct {
	Name      string `json:"name"`
	Target    string `json:"target"`
	Transport string `json:"transport"`
	AuthType  string `json:"auth_type"`
}

type contextsResponse struct {
	Active   string        `json:"active"`
	Contexts []contextInfo `json:"contexts"`
}

// ContextsList returns configured daemon contexts and the active context.
func ContextsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		contexts := store.List()
		out := make([]contextInfo, 0, len(contexts))
		for _, ctx := range contexts {
			out = append(out, contextInfo{
				Name:      ctx.Name,
				Target:    ctx.Target,
				Transport: ctx.Transport,
				AuthType:  ctx.Auth.Type,
			})
		}

		writeJSON(w, http.StatusOK, contextsResponse{
			Active:   store.ActiveName(),
			Contexts: out,
		})
	}
}

// ContextUse switches the active daemon context.
func ContextUse() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Use(req.Name); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"active": req.Name,
		})
	}
}
