package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/template"
)

// templateStore returns the daemon-owned local template library, ensuring
// the built-ins are installed.
func templateStore() *template.Store {
	s := template.NewStore("")
	_ = s.InstallDefaults()
	return s
}

// ListTemplates handles GET /v1/templates.
func ListTemplates(w http.ResponseWriter, r *http.Request) {
	s := templateStore()
	names, err := s.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		t, err := s.Get(name)
		if err != nil {
			continue
		}
		src := string(template.SourceBuiltin)
		if t.Source != nil && t.Source.Type != "" {
			src = string(t.Source.Type)
		}
		items = append(items, map[string]any{
			"name":    t.Name,
			"version": t.Version,
			"summary": t.Summary,
			"source":  src,
			"tags":    t.Tags,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(items), "templates": items})
}

// GetTemplate handles GET /v1/templates/{name}.
func GetTemplate(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	t, err := templateStore().Get(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"template":      t,
		"image":         template.ResolveTemplateImage(t),
		"contentDigest": t.ContentDigest(),
		"problems":      t.Validate(),
	})
}

// ForkTemplate handles POST /v1/templates/fork — {source, name}.
func ForkTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Source == "" || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source and name are required"})
		return
	}
	forked, err := templateStore().Fork(req.Source, req.Name)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"name": forked.Name, "source": forked.Source})
}

// ValidateTemplate handles POST /v1/templates/validate — either
// {name: "<installed>"} or {template: {...}}.
func ValidateTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string             `json:"name"`
		Template *template.Template `json:"template"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	t := req.Template
	if t == nil {
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "one of name or template is required"})
			return
		}
		loaded, err := templateStore().Get(req.Name)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
			return
		}
		t = loaded
	}

	problems := t.Validate()
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":    len(problems) == 0,
		"problems": problems,
	})
}
