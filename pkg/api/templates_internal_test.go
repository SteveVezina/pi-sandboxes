package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func templateTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestListTemplates_ReturnsBuiltins(t *testing.T) {
	templateTestHome(t)
	w := httptest.NewRecorder()
	ListTemplates(w, httptest.NewRequest(http.MethodGet, "/v1/templates", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code %d", w.Code)
	}
	var resp struct {
		Count     int              `json:"count"`
		Templates []map[string]any `json:"templates"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Count != 7 {
		t.Fatalf("want 7 built-ins, got %d", resp.Count)
	}
}

func TestGetTemplate_IncludesDigestAndImage(t *testing.T) {
	templateTestHome(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/templates/node", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "node"})
	w := httptest.NewRecorder()
	GetTemplate(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code %d: %s", w.Code, w.Body)
	}
	body := w.Body.String()
	if !strings.Contains(body, "contentDigest") || !strings.Contains(body, "sha256:") {
		t.Fatalf("missing digest: %s", body)
	}
	if !strings.Contains(body, "docker.io/library/") {
		t.Fatalf("missing resolved image: %s", body)
	}
}

func TestForkTemplate_ThenValidate(t *testing.T) {
	templateTestHome(t)

	fw := httptest.NewRecorder()
	ForkTemplate(fw, httptest.NewRequest(http.MethodPost, "/v1/templates/fork",
		strings.NewReader(`{"source":"python","name":"my-python"}`)))
	if fw.Code != http.StatusCreated {
		t.Fatalf("fork code %d: %s", fw.Code, fw.Body)
	}

	// Second fork onto the same name -> 409.
	fw2 := httptest.NewRecorder()
	ForkTemplate(fw2, httptest.NewRequest(http.MethodPost, "/v1/templates/fork",
		strings.NewReader(`{"source":"python","name":"my-python"}`)))
	if fw2.Code != http.StatusConflict {
		t.Fatalf("duplicate fork want 409, got %d", fw2.Code)
	}

	vw := httptest.NewRecorder()
	ValidateTemplate(vw, httptest.NewRequest(http.MethodPost, "/v1/templates/validate",
		strings.NewReader(`{"name":"my-python"}`)))
	var vr struct {
		Valid    bool     `json:"valid"`
		Problems []string `json:"problems"`
	}
	json.Unmarshal(vw.Body.Bytes(), &vr)
	if !vr.Valid {
		t.Fatalf("forked template should be valid: %v", vr.Problems)
	}
}

func TestValidateTemplate_InlineBadDefinition(t *testing.T) {
	templateTestHome(t)
	w := httptest.NewRecorder()
	ValidateTemplate(w, httptest.NewRequest(http.MethodPost, "/v1/templates/validate",
		strings.NewReader(`{"template":{"name":"","network":"bogus"}}`)))
	var vr struct {
		Valid    bool     `json:"valid"`
		Problems []string `json:"problems"`
	}
	json.Unmarshal(w.Body.Bytes(), &vr)
	if vr.Valid || len(vr.Problems) == 0 {
		t.Fatalf("expected invalid with problems, got %+v", vr)
	}
}
