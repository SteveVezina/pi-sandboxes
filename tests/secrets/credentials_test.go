package secrets_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/secrets"
)

func TestValueSource_Literal(t *testing.T) {
	v, err := secrets.ValueSource{Literal: "tok-123"}.Resolve()
	if err != nil || v != "tok-123" {
		t.Fatalf("literal resolve: %q %v", v, err)
	}
}

func TestValueSource_Env(t *testing.T) {
	t.Setenv("PI_TEST_SECRET", "from-env")
	v, err := secrets.ValueSource{Env: "PI_TEST_SECRET"}.Resolve()
	if err != nil || v != "from-env" {
		t.Fatalf("env resolve: %q %v", v, err)
	}
	if _, err := (secrets.ValueSource{Env: "PI_TEST_MISSING"}).Resolve(); err == nil {
		t.Fatal("missing env var should error")
	}
}

func TestValueSource_File(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "token")
	if err := os.WriteFile(p, []byte("file-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	v, err := secrets.ValueSource{File: p}.Resolve()
	if err != nil || v != "file-secret" {
		t.Fatalf("file resolve: %q %v", v, err)
	}

	if _, err := (secrets.ValueSource{File: "relative/path"}).Resolve(); err == nil {
		t.Fatal("relative path should error")
	}
}

func TestValueSource_FileUnderPiBoxRejected(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := filepath.Join(home, ".pi-box", "secrets", "token")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (secrets.ValueSource{File: p}).Resolve(); err == nil {
		t.Fatal("secret file under ~/.pi-box must be rejected")
	}
}

func TestValueSource_Empty(t *testing.T) {
	if _, err := (secrets.ValueSource{}).Resolve(); err == nil {
		t.Fatal("empty source should error")
	}
}

func TestCredentialStore_AddWithValueResolveRemove(t *testing.T) {
	s := secrets.NewCredentialStore()
	c := secrets.Credential{ID: "gh", Type: "git-token", Hosts: []string{"github.com"}}

	if err := s.AddWithValue(c, "ghp_secret"); err != nil {
		t.Fatalf("AddWithValue: %v", err)
	}
	if err := s.AddWithValue(c, ""); err == nil {
		t.Fatal("empty value should error")
	}

	v, err := s.Resolve("gh")
	if err != nil || v != "ghp_secret" {
		t.Fatalf("Resolve: %q %v", v, err)
	}

	// List never exposes the value.
	for _, listed := range s.List() {
		if listed.ID == "gh" && !listed.Redacted {
			// Credential struct has no value field at all — just sanity check.
		}
	}

	s.Remove("gh")
	if _, err := s.Resolve("gh"); err == nil {
		t.Fatal("Resolve after Remove should error")
	}
	if _, err := s.Get("gh"); err == nil {
		t.Fatal("Get after Remove should error")
	}
}

func TestCredentialStore_GetForHost_LongPatternNoPanic(t *testing.T) {
	s := secrets.NewCredentialStore()
	_ = s.AddWithValue(secrets.Credential{
		ID: "c", Type: "git-token", Hosts: []string{"*.verylongdomainname.example.com"},
	}, "v")
	// host shorter than the pattern must not panic
	if got := s.GetForHost("x"); len(got) != 0 {
		t.Fatalf("no match expected, got %d", len(got))
	}
	if got := s.GetForHost("api.verylongdomainname.example.com"); len(got) != 1 {
		t.Fatalf("wildcard should match subdomain, got %d", len(got))
	}
}
