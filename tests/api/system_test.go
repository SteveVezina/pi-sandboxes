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

func TestSystemRuntimes_ReturnsCapabilityReports(t *testing.T) {
	store, _ := newTestStore(t)
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/system/runtimes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Backends []map[string]interface{} `json:"backends"`
		Best     string                   `json:"best"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Backends) != 4 {
		t.Fatalf("expected 4 capability reports, got %d", len(body.Backends))
	}
	for _, backend := range body.Backends {
		if _, has := backend["security_level"]; has {
			t.Errorf("backend %v still exposes security_level; capability reports replaced it (PROP-008)", backend["mode"])
		}
		if _, has := backend["isolation_tier"]; !has {
			t.Errorf("backend %v missing isolation_tier", backend["mode"])
		}
		if _, has := backend["compat_tier"]; !has {
			t.Errorf("backend %v missing compat_tier", backend["mode"])
		}
		if avail, _ := backend["available"].(bool); !avail {
			if reason, _ := backend["reason"].(string); reason == "" {
				t.Errorf("unavailable backend %v must carry a reason", backend["mode"])
			}
		}
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
