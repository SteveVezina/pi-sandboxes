package network_test

import (
	"encoding/base64"
	"testing"

	"github.com/pi-sandbox/pi/pkg/network"
	"github.com/pi-sandbox/pi/pkg/secrets"
)

func TestCredentialInjectorFromStore_GitToken(t *testing.T) {
	s := secrets.NewCredentialStore()
	if err := s.AddWithValue(secrets.Credential{
		ID: "gh", Type: "git-token", Hosts: []string{"github.com"},
	}, "ghp_x"); err != nil {
		t.Fatal(err)
	}

	inj := network.CredentialInjectorFromStore(s)
	got := inj("github.com")
	if len(got) != 1 || got[0].Name != "Authorization" {
		t.Fatalf("got %+v", got)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:ghp_x"))
	if got[0].Value != want {
		t.Fatalf("value = %q, want %q", got[0].Value, want)
	}

	if len(inj("evil.example.com")) != 0 {
		t.Error("no injection for non-matching host")
	}
}

func TestCredentialInjectorFromStore_RegistryAuth(t *testing.T) {
	s := secrets.NewCredentialStore()
	_ = s.AddWithValue(secrets.Credential{
		ID: "npm", Name: "ci-bot", Type: "registry-auth", Hosts: []string{"*.npmjs.org"},
	}, "npm_tok")

	got := network.CredentialInjectorFromStore(s)("registry.npmjs.org")
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("ci-bot:npm_tok"))
	if len(got) != 1 || got[0].Value != want {
		t.Fatalf("got %+v want %q", got, want)
	}
}

func TestCredentialInjectorFromStore_NilStore(t *testing.T) {
	if network.CredentialInjectorFromStore(nil)("github.com") != nil {
		t.Fatal("nil store must yield nil injections")
	}
}
