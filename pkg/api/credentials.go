package api

import (
	"encoding/json"
	"net/http"

	"github.com/pi-sandbox/pi/pkg/secrets"
)

// credentialStore holds credential injection rules and their in-memory
// values for the egress proxy (ADR-006 / F30 T30.7). Daemon-singleton;
// nothing is persisted to disk.
var credentialStore = secrets.NewCredentialStore()

// CredentialStoreInstance returns the process credential store so the
// daemon egress proxy can resolve values at request time (T30.8).
func CredentialStoreInstance() *secrets.CredentialStore { return credentialStore }

// credentialRequest is the POST /v1/credentials body.
type credentialRequest struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Type        string              `json:"type"`     // git-token, registry-auth, ...
	Hosts       []string            `json:"hosts"`    // exact host or "*.suffix"
	InjectAs    string              `json:"injectAs"` // header, env, file
	ValueSource secrets.ValueSource `json:"valueFrom"`
	// Value is a shorthand for valueFrom.value.
	Value string `json:"value"`
}

// credentialView is the redacted representation returned by GET.
type credentialView struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Hosts    []string `json:"hosts"`
	InjectAs string   `json:"injectAs"`
	Value    string   `json:"value"` // always "[redacted]"
}

// RegisterCredential handles POST /v1/credentials.
func RegisterCredential(w http.ResponseWriter, r *http.Request) {
	var req credentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.ID == "" || req.Type == "" || len(req.Hosts) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id, type, and hosts are required"})
		return
	}

	src := req.ValueSource
	if req.Value != "" && src.Literal == "" {
		src.Literal = req.Value
	}
	value, err := src.Resolve()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cred := secrets.Credential{
		ID:       req.ID,
		Name:     req.Name,
		Type:     req.Type,
		Hosts:    req.Hosts,
		InjectAs: req.InjectAs,
		Redacted: true,
	}
	if err := credentialStore.AddWithValue(cred, value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": req.ID})
}

// ListCredentials handles GET /v1/credentials — rules only, values redacted.
func ListCredentials(w http.ResponseWriter, r *http.Request) {
	creds := credentialStore.List()
	views := make([]credentialView, 0, len(creds))
	for _, c := range creds {
		views = append(views, credentialView{
			ID:       c.ID,
			Name:     c.Name,
			Type:     c.Type,
			Hosts:    c.Hosts,
			InjectAs: c.InjectAs,
			Value:    "[redacted]",
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":       len(views),
		"credentials": views,
	})
}
