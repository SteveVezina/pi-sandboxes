package daemon

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func newAuthTestRouter(t *testing.T, token string) http.Handler {
	t.Helper()
	store := sandbox.NewStore(filepath.Join(t.TempDir(), "sandboxes"))
	return newRouterWithAuth(token, store)
}

func TestBearerAuth_NoTokenConfigured_PassesThrough(t *testing.T) {
	r := newAuthTestRouter(t, "")
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("no token configured should not 401, got %d", w.Code)
	}
}

func TestBearerAuth_MissingToken_401(t *testing.T) {
	r := newAuthTestRouter(t, "s3cret")
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without Authorization header, got %d", w.Code)
	}
}

func TestBearerAuth_WrongToken_401(t *testing.T) {
	r := newAuthTestRouter(t, "s3cret")
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes", nil)
	req.Header.Set("Authorization", "Bearer nope")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 with wrong token, got %d", w.Code)
	}
}

func TestBearerAuth_ValidToken_PassesThrough(t *testing.T) {
	r := newAuthTestRouter(t, "s3cret")
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("valid token should pass, got 401")
	}
}

func TestBearerAuth_HealthAndOptionsExempt(t *testing.T) {
	r := newAuthTestRouter(t, "s3cret")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/health must be reachable without a token, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/v1/sandboxes", nil))
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("OPTIONS pre-flight must not 401")
	}
}

func TestDaemon_PublicBindWithoutToken_FailsClosed(t *testing.T) {
	store := sandbox.NewStore(filepath.Join(t.TempDir(), "sandboxes"))
	d := New(filepath.Join(t.TempDir(), "d.sock"), 8080, store)
	d.SetHTTPHost("0.0.0.0")
	if err := d.Start(); err == nil {
		t.Fatal("expected Start to refuse a non-loopback bind without PI_DAEMON_TOKEN")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	for _, tc := range []struct {
		host string
		want bool
	}{
		{"", true},
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"0.0.0.0", false},
		{"10.0.0.5", false},
	} {
		if got := isLoopbackHost(tc.host); got != tc.want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}
