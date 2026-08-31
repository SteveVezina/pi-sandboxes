package network_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/network"
)

func TestPolicyFor_DefaultRestricted_AllowsRegistriesDeniesRest(t *testing.T) {
	p, err := network.PolicyFor("restricted", nil)
	if err != nil {
		t.Fatalf("PolicyFor: %v", err)
	}
	if !p.IsAllowed("github.com") {
		t.Error("restricted should allow github.com")
	}
	if !p.IsAllowed("registry.npmjs.org") {
		t.Error("restricted should allow registry.npmjs.org")
	}
	if p.IsAllowed("evil.example.com") {
		t.Error("restricted should deny non-allowlisted host")
	}
}

func TestPolicyFor_NoneMode_DeniesEverything(t *testing.T) {
	p, err := network.PolicyFor("none", []string{"github.com"})
	if err != nil {
		t.Fatalf("PolicyFor: %v", err)
	}
	for _, host := range []string{"github.com", "example.com", "169.254.169.254"} {
		if p.IsAllowed(host) {
			t.Errorf("none should deny %s", host)
		}
	}
}

func TestPolicyFor_OpenMode_AllowsHostsButNotDefaultDeny(t *testing.T) {
	p, err := network.PolicyFor("open", nil)
	if err != nil {
		t.Fatalf("PolicyFor: %v", err)
	}
	if !p.IsAllowed("anything.example.com") {
		t.Error("open should allow arbitrary hosts")
	}
	if p.IsAllowed("169.254.169.254") {
		t.Error("open must still deny the cloud metadata endpoint (ADR-006 §4)")
	}
}

func TestPolicyFor_ExtraAllow_WidensRestrictedOnly(t *testing.T) {
	p, err := network.PolicyFor("restricted", []string{"internal.corp", "  ", ""})
	if err != nil {
		t.Fatalf("PolicyFor: %v", err)
	}
	if !p.IsAllowed("internal.corp") {
		t.Error("extra allow host should be permitted in restricted mode")
	}
	if !p.IsAllowed("api.internal.corp") {
		t.Error("subdomain of extra allow host should be permitted")
	}
}

func TestPolicyFor_ExtraAllowCannotRelaxDefaultDeny(t *testing.T) {
	p, err := network.PolicyFor("restricted", []string{"169.254.169.254"})
	if err != nil {
		t.Fatalf("PolicyFor: %v", err)
	}
	if p.IsAllowed("169.254.169.254") {
		t.Error("default deny must win even when the caller allowlists the host")
	}
}

func TestPolicyFor_InvalidMode_Errors(t *testing.T) {
	if _, err := network.PolicyFor("wibble", nil); err == nil {
		t.Fatal("expected error for invalid network mode")
	}
}

func TestValidMode(t *testing.T) {
	for _, m := range []string{"none", "restricted", "open"} {
		if !network.ValidMode(m) {
			t.Errorf("%s should be valid", m)
		}
	}
	if network.ValidMode("bridge") {
		t.Error("bridge is not a valid network mode")
	}
}
