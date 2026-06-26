package secrets_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/secrets"
)

func TestBroker_AddAndGet(t *testing.T) {
	b := secrets.NewBroker()
	b.Add("github-token", "ghp_abc123", []string{"git"}, "token")

	s, ok := b.Get("github-token")
	if !ok {
		t.Fatal("Expected secret to exist")
	}
	if s.Name != "github-token" {
		t.Errorf("Expected name 'github-token', got '%s'", s.Name)
	}
}

func TestBroker_Has(t *testing.T) {
	b := secrets.NewBroker()
	b.Add("token1", "value1", nil, "token")

	if !b.Has("token1") {
		t.Error("Expected token1 to exist")
	}
	if b.Has("nonexistent") {
		t.Error("Expected nonexistent to not exist")
	}
}

func TestBroker_Remove(t *testing.T) {
	b := secrets.NewBroker()
	b.Add("token1", "value1", nil, "token")

	b.Remove("token1")
	if b.Has("token1") {
		t.Error("Expected token1 to be removed")
	}
}

func TestBroker_List(t *testing.T) {
	b := secrets.NewBroker()
	b.Add("token1", "v1", nil, "token")
	b.Add("token2", "v2", nil, "token")

	names := b.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(names))
	}
}

func TestSSHAgent_CanForward(t *testing.T) {
	agent := secrets.DefaultSSHAgent()
	if agent.CanForward("git") {
		t.Error("Default agent should not forward (Forward=false)")
	}

	agent.Forward = true
	if !agent.CanForward("git") {
		t.Error("Agent should forward for git")
	}
	if agent.CanForward("bash") {
		t.Error("Agent should not forward for bash (scoped to git)")
	}
}

func TestSSHAgent_Scopes(t *testing.T) {
	agent := &secrets.SSHAgent{
		Forward: true,
		Scopes:  []string{"git", "gh"},
	}

	if !agent.CanForward("git") {
		t.Error("Should forward for git")
	}
	if !agent.CanForward("gh") {
		t.Error("Should forward for gh")
	}
	if agent.CanForward("curl") {
		t.Error("Should not forward for curl")
	}
}

func TestTokenHelper_Scoped(t *testing.T) {
	helper := &secrets.TokenHelper{
		Enabled: true,
		Scopes:  []string{"github.com"},
	}

	if !helper.IsScoped("https://github.com/repo.git") {
		t.Error("Should allow github.com URLs")
	}
	if helper.IsScoped("https://evil.com/repo.git") {
		t.Error("Should not allow evil.com URLs")
	}
}

func TestTokenHelper_Disabled(t *testing.T) {
	helper := &secrets.TokenHelper{
		Enabled: false,
		Scopes:  []string{"github.com"},
	}

	if helper.IsScoped("https://github.com/repo.git") {
		t.Error("Disabled helper should not allow any URLs")
	}
}

func TestTokenHelper_EmptyScopes(t *testing.T) {
	helper := &secrets.TokenHelper{
		Enabled: true,
		Scopes:  nil, // Empty = all hosts
	}

	if !helper.IsScoped("https://any-host.com/repo.git") {
		t.Error("Empty scopes should allow all hosts")
	}
}
