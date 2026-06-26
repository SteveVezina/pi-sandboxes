package network_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/network"
)

func TestModeNone(t *testing.T) {
	m := network.ModeNone
	if m.IsAllowed("github.com") {
		t.Error("ModeNone should block all hosts")
	}
	if m.IsAllowed("169.254.169.254") {
		t.Error("ModeNone should block metadata endpoint")
	}
}

func TestModeOpen(t *testing.T) {
	m := network.ModeOpen
	if !m.IsAllowed("github.com") {
		t.Error("ModeOpen should allow all hosts")
	}
	if !m.IsAllowed("169.254.169.254") {
		t.Error("ModeOpen should allow even metadata (user opted in)")
	}
}

func TestModeRestricted(t *testing.T) {
	m := network.ModeRestricted
	if !m.IsAllowed("github.com") {
		t.Error("ModeRestricted should allow github.com")
	}
	if !m.IsAllowed("api.github.com") {
		t.Error("ModeRestricted should allow api.github.com")
	}
	if m.IsAllowed("evil.com") {
		t.Error("ModeRestricted should block evil.com")
	}
	if m.IsAllowed("169.254.169.254") {
		t.Error("ModeRestricted should block metadata endpoint")
	}
}

func TestDefaultDeny(t *testing.T) {
	if !network.IsDefaultDeny("169.254.169.254") {
		t.Error("169.254.169.254 should be in default deny")
	}
	if !network.IsDefaultDeny("localhost") {
		t.Error("localhost should be in default deny")
	}
	if !network.IsDefaultDeny("127.0.0.1") {
		t.Error("127.0.0.1 should be in default deny")
	}
	if network.IsDefaultDeny("github.com") {
		t.Error("github.com should NOT be in default deny")
	}
}

func TestDomainList_SubdomainMatch(t *testing.T) {
	dl := network.DomainList{"example.com"}
	if !dl.Contains("api.example.com") {
		t.Error("api.example.com should match example.com")
	}
	if !dl.Contains("sub.api.example.com") {
		t.Error("sub.api.example.com should match example.com")
	}
	if dl.Contains("notexample.com") {
		t.Error("notexample.com should NOT match example.com")
	}
}

func TestPolicy_Default(t *testing.T) {
	p := network.DefaultPolicy()
	if p.Mode != network.ModeRestricted {
		t.Errorf("Expected mode 'restricted', got '%s'", p.Mode)
	}
	if p.Validate() != nil {
		t.Error("Default policy should be valid")
	}
}

func TestPolicy_IsAllowed(t *testing.T) {
	p := network.DefaultPolicy()
	if !p.IsAllowed("github.com") {
		t.Error("Default policy should allow github.com")
	}
	if p.IsAllowed("169.254.169.254") {
		t.Error("Default policy should block metadata endpoint")
	}
	if p.IsAllowed("evil.com") {
		t.Error("Default policy should block unknown domains")
	}
}

func TestPolicy_InvalidMode(t *testing.T) {
	p := &network.Policy{Mode: "invalid"}
	if err := p.Validate(); err == nil {
		t.Error("Expected error for invalid mode")
	}
}

func TestPolicy_ApplyNetworkMode(t *testing.T) {
	p := network.DefaultPolicy()
	pNone := p.ApplyNetworkMode(network.ModeNone)
	if pNone.Mode != network.ModeNone {
		t.Error("Expected mode to change to none")
	}
	if pNone.IsAllowed("github.com") {
		t.Error("ModeNone should block all")
	}

	pOpen := p.ApplyNetworkMode(network.ModeOpen)
	if pOpen.Mode != network.ModeOpen {
		t.Error("Expected mode to change to open")
	}
	if !pOpen.IsAllowed("github.com") {
		t.Error("ModeOpen should allow all")
	}
}

func TestPolicy_DenyOverridesAllow(t *testing.T) {
	p := network.DefaultPolicy()
	// github.com is in allowlist, but 169.254.169.254 is in denylist
	if p.IsAllowed("169.254.169.254") {
		t.Error("Deny list should override allowlist")
	}
}
