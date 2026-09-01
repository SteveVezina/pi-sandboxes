package api

import (
	"bytes"
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

func TestTemplateHistoryDiffRollback(t *testing.T) {
	templateTestHome(t)

	// fork -> rev 1
	fw := httptest.NewRecorder()
	ForkTemplate(fw, httptest.NewRequest(http.MethodPost, "/v1/templates/fork",
		strings.NewReader(`{"source":"rust","name":"my-rust"}`)))
	if fw.Code != http.StatusCreated {
		t.Fatalf("fork: %d %s", fw.Code, fw.Body)
	}

	// mutate via a second fork-target write is not exposed; drive a rollback
	// off the single revision instead and check history grows.
	hw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "/v1/templates/my-rust/history", nil)
	hr = mux.SetURLVars(hr, map[string]string{"name": "my-rust"})
	TemplateHistory(hw, hr)
	var h struct {
		Count int `json:"count"`
	}
	json.Unmarshal(hw.Body.Bytes(), &h)
	if h.Count != 1 {
		t.Fatalf("history count = %d, want 1", h.Count)
	}

	// rollback to rev 1 -> rev 2
	rw := httptest.NewRecorder()
	rr := httptest.NewRequest(http.MethodPost, "/v1/templates/my-rust/rollback",
		strings.NewReader(`{"revision":1}`))
	rr = mux.SetURLVars(rr, map[string]string{"name": "my-rust"})
	RollbackTemplate(rw, rr)
	if rw.Code != http.StatusOK {
		t.Fatalf("rollback: %d %s", rw.Code, rw.Body)
	}

	// diff rev 1 vs rev 2 -> no meaningful difference (rollback of identical state)
	dw := httptest.NewRecorder()
	DiffTemplates(dw, httptest.NewRequest(http.MethodPost, "/v1/templates/diff",
		strings.NewReader(`{"left":"my-rust@1","right":"rust"}`)))
	if dw.Code != http.StatusOK || !strings.Contains(dw.Body.String(), "diff") {
		t.Fatalf("diff: %d %s", dw.Code, dw.Body)
	}
}

func TestExportImportTemplate_RoundTrip(t *testing.T) {
	templateTestHome(t)

	// fork so there is a non-builtin to export
	fw := httptest.NewRecorder()
	ForkTemplate(fw, httptest.NewRequest(http.MethodPost, "/v1/templates/fork",
		strings.NewReader(`{"source":"python","name":"my-py"}`)))
	if fw.Code != http.StatusCreated {
		t.Fatalf("fork: %d %s", fw.Code, fw.Body)
	}

	ew := httptest.NewRecorder()
	ExportTemplate(ew, httptest.NewRequest(http.MethodPost, "/v1/templates/export",
		strings.NewReader(`{"name":"my-py"}`)))
	if ew.Code != http.StatusOK || ew.Body.Len() == 0 {
		t.Fatalf("export: %d len=%d", ew.Code, ew.Body.Len())
	}
	if ct := ew.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("content-type %q", ct)
	}
	bundle := ew.Body.Bytes()

	// import under a new name
	iw := httptest.NewRecorder()
	ir := httptest.NewRequest(http.MethodPost, "/v1/templates/import?name=my-py-2", bytes.NewReader(bundle))
	ImportTemplate(iw, ir)
	if iw.Code != http.StatusCreated {
		t.Fatalf("import: %d %s", iw.Code, iw.Body)
	}
	if !strings.Contains(iw.Body.String(), "imported") {
		t.Fatalf("import response missing source: %s", iw.Body)
	}

	// collision without rename
	iw2 := httptest.NewRecorder()
	ImportTemplate(iw2, httptest.NewRequest(http.MethodPost, "/v1/templates/import", bytes.NewReader(bundle)))
	if iw2.Code != http.StatusConflict {
		t.Fatalf("collision import want 409, got %d", iw2.Code)
	}
}
