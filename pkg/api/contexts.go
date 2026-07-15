package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	pictx "github.com/pi-sandbox/pi/pkg/context"
)

type contextInfo struct {
	Name      string `json:"name"`
	Target    string `json:"target"`
	Transport string `json:"transport"`
	AuthType  string `json:"auth_type"`
	TokenEnv  string `json:"token_env,omitempty"`
	SSHUser   string `json:"ssh_user,omitempty"`
	SSHHost   string `json:"ssh_host,omitempty"`
}

type contextsResponse struct {
	Active   string        `json:"active"`
	Contexts []contextInfo `json:"contexts"`
}

type contextRequest struct {
	Name      string `json:"name"`
	Target    string `json:"target"`
	Transport string `json:"transport"`
	AuthType  string `json:"auth_type"`
	TokenEnv  string `json:"token_env"`
	SSHUser   string `json:"ssh_user"`
	SSHHost   string `json:"ssh_host"`
	Token     string `json:"token"`
	Bearer    string `json:"bearer_token"`
	Key       string `json:"key"`
	Private   string `json:"private_key"`
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
				TokenEnv:  ctx.Auth.TokenEnv,
				SSHUser:   ctx.Auth.SSHUser,
				SSHHost:   ctx.Auth.SSHHost,
			})
		}

		writeJSON(w, http.StatusOK, contextsResponse{
			Active:   store.ActiveName(),
			Contexts: out,
		})
	}
}

// ContextCreate creates a daemon context through the GUI/API.
func ContextCreate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req contextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if err := validateContextRequest(req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		ctx := contextFromRequest(req)
		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Create(ctx); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, contextToInfo(ctx))
	}
}

// ContextGet returns one context without raw credential material.
func ContextGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		ctx, err := store.Get(name)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, contextToInfo(ctx))
	}
}

// ContextUpdate updates an existing non-local context.
func ContextUpdate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		var req contextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if err := validateContextRequest(req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Name == "" {
			req.Name = name
		}
		ctx := contextFromRequest(req)
		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Update(name, ctx); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, contextToInfo(ctx))
	}
}

// ContextDelete deletes a non-local context.
func ContextDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if err := store.Delete(name); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"deleted": name,
			"active":  store.ActiveName(),
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

func contextFromRequest(req contextRequest) pictx.Context {
	authType := req.AuthType
	if authType == "" {
		authType = inferContextAuth(req.Transport)
	}
	return pictx.Context{
		Name:      req.Name,
		Target:    req.Target,
		Transport: req.Transport,
		Auth: pictx.AuthConfig{
			Type:     authType,
			TokenEnv: req.TokenEnv,
			SSHUser:  req.SSHUser,
			SSHHost:  req.SSHHost,
		},
	}
}

func validateContextRequest(req contextRequest) error {
	if req.Token != "" || req.Bearer != "" {
		return fmt.Errorf("raw bearer tokens are not accepted; set token_env to an environment variable name")
	}
	if req.Key != "" || req.Private != "" {
		return fmt.Errorf("raw SSH key material is not accepted; use ssh-agent authentication")
	}
	return nil
}

func contextToInfo(ctx pictx.Context) contextInfo {
	return contextInfo{
		Name:      ctx.Name,
		Target:    ctx.Target,
		Transport: ctx.Transport,
		AuthType:  ctx.Auth.Type,
		TokenEnv:  ctx.Auth.TokenEnv,
		SSHUser:   ctx.Auth.SSHUser,
		SSHHost:   ctx.Auth.SSHHost,
	}
}

func inferContextAuth(transport string) string {
	switch transport {
	case pictx.TransportHTTP:
		return pictx.AuthBearerToken
	case pictx.TransportSSH:
		return pictx.AuthSSHAgent
	default:
		return pictx.AuthNone
	}
}
