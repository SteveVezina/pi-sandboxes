package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/daemon"
)

func TestSystemStatusEndpoint(t *testing.T) {
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/system/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["daemon"] != "connected" {
		t.Fatalf("expected connected daemon status, got %v", body["daemon"])
	}
}

func TestSystemDoctorEndpoint(t *testing.T) {
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/system/doctor", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Issues") {
		t.Fatalf("expected doctor issues in response, got %s", w.Body.String())
	}
}

func TestSupportBundleRedactsHomePath(t *testing.T) {
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/support-bundle", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["redacted"] != true {
		t.Fatalf("expected redacted flag")
	}
	if strings.Contains(w.Body.String(), "/Users/") {
		t.Fatalf("expected support bundle to redact home paths, got %s", w.Body.String())
	}
	config := body["config"].(map[string]interface{})
	if path, _ := config["pi_home"].(string); strings.HasPrefix(path, "/Users/") {
		t.Fatalf("expected pi_home to be redacted, got %s", path)
	}
}
