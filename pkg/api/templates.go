package api

import (
	"encoding/json"
	"io"
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

// TemplateHistory handles GET /v1/templates/{name}/history.
func TemplateHistory(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	revs, err := templateStore().History(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "count": len(revs), "revisions": revs})
}

// DiffTemplates handles POST /v1/templates/diff — {left, right} refs
// ("name" or "name@N").
func DiffTemplates(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Left  string `json:"left"`
		Right string `json:"right"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Left == "" || req.Right == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "left and right refs are required"})
		return
	}
	s := templateStore()
	l, err := s.ResolveRef(req.Left)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	rt, err := s.ResolveRef(req.Right)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"left": req.Left, "right": req.Right, "diff": template.Diff(l, rt)})
}

// PromoteTemplate handles POST /v1/templates/{name}/promote — {default}.
func PromoteTemplate(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var req struct {
		Default bool `json:"default"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if !req.Default {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "nothing to promote (set default: true)"})
		return
	}
	if err := templateStore().SetDefault(name); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "default": true})
}

// ExportTemplate handles POST /v1/templates/export — {name}. Returns the
// OCI image-layout bundle tar (ADR-008).
func ExportTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	t, err := templateStore().Get(req.Name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "template not found"})
		return
	}
	bundle, err := template.ExportBundle(t)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+req.Name+`.oci.tar"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bundle)
}

// ImportTemplate handles POST /v1/templates/import — the raw bundle tar as
// the request body, optional ?name= to rename.
func ImportTemplate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20)) // 32 MiB cap
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body: " + err.Error()})
		return
	}
	imported, err := templateStore().Import(body, r.URL.Query().Get("name"))
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"name":          imported.Name,
		"source":        imported.Source,
		"contentDigest": imported.ContentDigest(),
	})
}

// RollbackTemplate handles POST /v1/templates/{name}/rollback — {revision}.
func RollbackTemplate(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	var req struct {
		Revision int `json:"revision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Revision < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "revision (>= 1) is required"})
		return
	}
	restored, err := templateStore().Rollback(name, req.Revision)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       name,
		"restoredTo": req.Revision,
		"generation": restored.Lineage.Generation,
	})
}
