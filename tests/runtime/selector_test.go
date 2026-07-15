package runtime_test

import (
	"context"
	"strings"
	"testing"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
)

// buildRegistry registers fake probers in priority order with the given
// availability per mode.
func buildRegistry(t *testing.T, available map[pruntime.Mode]bool) *pruntime.Registry {
	t.Helper()
	reg := pruntime.NewRegistry()
	tiers := map[pruntime.Mode]int{
		pruntime.ModeMicroVM: 4,
		pruntime.ModeSecure:  2,
		pruntime.ModeFast:    1,
		pruntime.ModeCompat:  1,
	}
	for _, m := range []pruntime.Mode{pruntime.ModeMicroVM, pruntime.ModeSecure, pruntime.ModeFast, pruntime.ModeCompat} {
		rep := pruntime.CapabilityReport{
			Mode:          string(m),
			Available:     available[m],
			IsolationTier: tiers[m],
		}
		if !available[m] {
			rep.Reason = string(m) + " prerequisites missing"
			rep.Missing = []string{string(m) + "-prereq"}
		}
		if err := reg.Register(&fakeProber{mode: m, report: rep}); err != nil {
			t.Fatalf("Register(%s): %v", m, err)
		}
	}
	return reg
}

func TestSelect_ExplicitAvailableModeHonored(t *testing.T) {
	reg := buildRegistry(t, map[pruntime.Mode]bool{pruntime.ModeCompat: true, pruntime.ModeSecure: true})

	sel, err := pruntime.Select(context.Background(), reg, pruntime.ModeCompat, pruntime.TrustTrusted, pruntime.FallbackPolicy{})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if sel.Resolved != pruntime.ModeCompat || sel.Fallback {
		t.Errorf("expected compat without fallback, got %+v", sel)
	}
}

func TestSelect_ExplicitUnavailableNoFallbackFailsWithGuidance(t *testing.T) {
	reg := buildRegistry(t, map[pruntime.Mode]bool{pruntime.ModeCompat: true})

	_, err := pruntime.Select(context.Background(), reg, pruntime.ModeSecure, pruntime.TrustTrusted, pruntime.FallbackPolicy{})
	if err == nil {
		t.Fatal("expected error for unavailable explicit mode without fallback")
	}
	if !strings.Contains(err.Error(), "secure-prereq") {
		t.Errorf("error must carry missing prerequisites for guidance, got: %v", err)
	}
}

func TestSelect_FallbackResolvesUpwardOnly(t *testing.T) {
	reg := buildRegistry(t, map[pruntime.Mode]bool{pruntime.ModeMicroVM: true, pruntime.ModeCompat: true})

	sel, err := pruntime.Select(context.Background(), reg, pruntime.ModeSecure, pruntime.TrustUntrusted, pruntime.FallbackPolicy{
		Allow: []pruntime.Mode{pruntime.ModeMicroVM, pruntime.ModeCompat},
	})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if sel.Resolved != pruntime.ModeMicroVM {
		t.Errorf("expected upward fallback to microvm, got %s", sel.Resolved)
	}
	if !sel.Fallback {
		t.Error("expected Fallback flag set")
	}
	if sel.Requested != pruntime.ModeSecure {
		t.Errorf("selection must preserve requested mode, got %s", sel.Requested)
	}
}

func TestSelect_NeverDowngradesBelowRequested(t *testing.T) {
	// Only compat (tier 1) available; secure (tier 2) requested; compat
	// allow-listed — still must fail: isolation never drops below request.
	reg := buildRegistry(t, map[pruntime.Mode]bool{pruntime.ModeCompat: true})

	_, err := pruntime.Select(context.Background(), reg, pruntime.ModeSecure, pruntime.TrustUntrusted, pruntime.FallbackPolicy{
		Allow: []pruntime.Mode{pruntime.ModeCompat},
	})
	if err == nil {
		t.Fatal("expected downgrade to compat to be rejected")
	}
}

func TestSelect_DenyWinsOverAllow(t *testing.T) {
	reg := buildRegistry(t, map[pruntime.Mode]bool{pruntime.ModeMicroVM: true})

	_, err := pruntime.Select(context.Background(), reg, pruntime.ModeSecure, pruntime.TrustUntrusted, pruntime.FallbackPolicy{
		Allow: []pruntime.Mode{pruntime.ModeMicroVM},
		Deny:  []pruntime.Mode{pruntime.ModeMicroVM},
	})
	if err == nil {
		t.Fatal("expected denied fallback candidate to be rejected")
	}
}

func TestSelect_AutoTrustedPrefersPerformance(t *testing.T) {
	reg := buildRegistry(t, map[pruntime.Mode]bool{
		pruntime.ModeMicroVM: true,
		pruntime.ModeSecure:  true,
		pruntime.ModeFast:    true,
		pruntime.ModeCompat:  true,
	})

	sel, err := pruntime.Select(context.Background(), reg, pruntime.ModeAuto, pruntime.TrustTrusted, pruntime.FallbackPolicy{})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if sel.Resolved != pruntime.ModeFast {
		t.Errorf("trusted auto must prefer performance (fast), got %s", sel.Resolved)
	}
}

func TestSelect_AutoUntrustedPrefersIsolation(t *testing.T) {
	reg := buildRegistry(t, map[pruntime.Mode]bool{
		pruntime.ModeSecure: true,
		pruntime.ModeFast:   true,
		pruntime.ModeCompat: true,
	})

	sel, err := pruntime.Select(context.Background(), reg, pruntime.ModeAuto, pruntime.TrustUntrusted, pruntime.FallbackPolicy{})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if sel.Resolved != pruntime.ModeSecure {
		t.Errorf("untrusted auto must prefer isolation (secure), got %s", sel.Resolved)
	}
}

func TestSelect_AutoUntrustedNeverPicksSharedKernelModes(t *testing.T) {
	// Only tier-1 shared-kernel modes available; untrusted auto must fail
	// rather than silently run untrusted code in fast/compat.
	reg := buildRegistry(t, map[pruntime.Mode]bool{pruntime.ModeFast: true, pruntime.ModeCompat: true})

	_, err := pruntime.Select(context.Background(), reg, pruntime.ModeAuto, pruntime.TrustUntrusted, pruntime.FallbackPolicy{})
	if err == nil {
		t.Fatal("expected untrusted auto with only shared-kernel modes to fail")
	}
}
