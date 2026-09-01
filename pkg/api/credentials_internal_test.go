package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/secrets"
)

func resetCredStore(t *testing.T) {
	t.Helper()
	credentialStore = secrets.NewCredentialStore()
	t.Cleanup(func() { credentialStore = secrets.NewCredentialStore() })
}

func postCred(body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	RegisterCredential(w, httptest.NewRequest(http.MethodPost, "/v1/credentials", strings.NewReader(body)))
	return w
}

func TestRegisterCredential_LiteralValue_StoredInMemoryRedactedInList(t *testing.T) {
	resetCredStore(t)

	w := postCred(`{"id":"gh","type":"git-token","hosts":["github.com"],"injectAs":"header","value":"ghp_secret"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body)
	}

	got, err := credentialStore.Resolve("gh")
	if err != nil || got != "ghp_secret" {
		t.Fatalf("resolve: %q %v", got, err)
	}

	lw := httptest.NewRecorder()
	ListCredentials(lw, httptest.NewRequest(http.MethodGet, "/v1/credentials", nil))
	var resp struct {
		Credentials []credentialView `json:"credentials"`
	}
	json.Unmarshal(lw.Body.Bytes(), &resp)
	if len(resp.Credentials) != 1 || resp.Credentials[0].Value != "[redacted]" {
		t.Fatalf("list must redact value: %s", lw.Body)
	}
	if strings.Contains(lw.Body.String(), "ghp_secret") {
		t.Fatalf("secret leaked in list response: %s", lw.Body)
	}
}

func TestRegisterCredential_EnvValue(t *testing.T) {
	resetCredStore(t)
	t.Setenv("PI_CRED_TEST", "env-secret")

	w := postCred(`{"id":"reg","type":"registry-auth","hosts":["registry.npmjs.org"],"valueFrom":{"env":"PI_CRED_TEST"}}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", w.Code, w.Body)
	}
	if v, _ := credentialStore.Resolve("reg"); v != "env-secret" {
		t.Fatalf("resolved %q", v)
	}
}

func TestRegisterCredential_MissingFields_400(t *testing.T) {
	resetCredStore(t)
	if w := postCred(`{"id":"x","value":"v"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for missing type/hosts, got %d", w.Code)
	}
	if w := postCred(`{"id":"x","type":"git-token","hosts":["h"]}`); w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for missing value, got %d", w.Code)
	}
}

func TestRegisterCredential_FileUnderPiBox_400(t *testing.T) {
	resetCredStore(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	w := postCred(`{"id":"x","type":"git-token","hosts":["h"],"valueFrom":{"file":"` + home + `/.pi-box/token"}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for secret file under ~/.pi-box, got %d: %s", w.Code, w.Body)
	}
}
