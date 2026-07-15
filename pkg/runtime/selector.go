package runtime

import (
	"context"
	"fmt"
	"strings"
)

// Trust classifies the workload for runtime selection (SPEC.md §14.7.5).
type Trust string

const (
	TrustTrusted   Trust = "trusted"
	TrustReviewed  Trust = "reviewed"
	TrustUntrusted Trust = "untrusted"
)

// FallbackPolicy is the explicit fallback configuration for a request.
// Deny always wins over Allow. An empty policy means: no fallback.
type FallbackPolicy struct {
	Allow []Mode `json:"allow,omitempty"`
	Deny  []Mode `json:"deny,omitempty"`
}

// Selection records a runtime resolution: what was requested, what
// actually resolved, and whether fallback occurred. Callers persist the
// requested-vs-resolved pair so fallback decisions stay visible.
type Selection struct {
	Requested Mode   `json:"requested"`
	Resolved  Mode   `json:"resolved"`
	Trust     Trust  `json:"trust"`
	Fallback  bool   `json:"fallback"`
	Reason    string `json:"reason,omitempty"`
}

// autoOrders defines trust-dependent auto resolution (SPEC.md §13, §14.7.5).
// Trusted work prefers performance; untrusted work prefers isolation and
// never resolves to a shared-kernel mode.
var autoOrders = map[Trust][]Mode{
	TrustTrusted:   {ModeFast, ModeCompat, ModeSecure, ModeIsolated, ModeMicroVM},
	TrustReviewed:  {ModeFast, ModeCompat, ModeSecure, ModeIsolated, ModeMicroVM},
	TrustUntrusted: {ModeSecure, ModeIsolated, ModeMicroVM},
}

// Select resolves a runtime mode from four separate inputs: requested
// mode, workload trust, discovered host capabilities, and explicit
// fallback policy. Isolation never silently drops below the requested
// mode; a denied fallback fails with actionable guidance.
func Select(ctx context.Context, reg *Registry, requested Mode, trust Trust, fallback FallbackPolicy) (Selection, error) {
	if trust == "" {
		trust = TrustTrusted
	}
	if requested == "" {
		requested = ModeAuto
	}

	if requested == ModeAuto {
		return selectAuto(ctx, reg, trust)
	}

	rep, ok := reg.Probe(ctx, requested)
	if !ok {
		return Selection{}, fmt.Errorf("unsupported runtime mode %s", requested)
	}
	if rep.Available {
		return Selection{Requested: requested, Resolved: requested, Trust: trust}, nil
	}

	sel, ferr := selectFallback(ctx, reg, requested, rep, trust, fallback)
	if ferr != nil {
		return Selection{}, ferr
	}
	return sel, nil
}

func selectAuto(ctx context.Context, reg *Registry, trust Trust) (Selection, error) {
	order := autoOrders[trust]
	var tried []string
	for _, mode := range order {
		rep, ok := reg.Probe(ctx, mode)
		if !ok {
			continue
		}
		if rep.Available {
			return Selection{Requested: ModeAuto, Resolved: mode, Trust: trust,
				Reason: fmt.Sprintf("auto (%s) resolved to %s", trust, mode)}, nil
		}
		tried = append(tried, string(mode))
	}
	return Selection{}, fmt.Errorf("no runtime available for %s workloads (tried: %s)", trust, strings.Join(tried, ", "))
}

func selectFallback(ctx context.Context, reg *Registry, requested Mode, requestedRep CapabilityReport, trust Trust, fallback FallbackPolicy) (Selection, error) {
	denied := map[Mode]bool{}
	for _, m := range fallback.Deny {
		denied[m] = true
	}

	baseErr := unavailableError(requested, requestedRep)
	if len(fallback.Allow) == 0 {
		return Selection{}, baseErr
	}

	var rejected []string
	for _, candidate := range fallback.Allow {
		if denied[candidate] {
			rejected = append(rejected, fmt.Sprintf("%s (denied by policy)", candidate))
			continue
		}
		rep, ok := reg.Probe(ctx, candidate)
		if !ok {
			rejected = append(rejected, fmt.Sprintf("%s (unknown mode)", candidate))
			continue
		}
		if rep.IsolationTier < requestedRep.IsolationTier {
			rejected = append(rejected, fmt.Sprintf("%s (isolation tier %d below requested %d)", candidate, rep.IsolationTier, requestedRep.IsolationTier))
			continue
		}
		if !rep.Available {
			rejected = append(rejected, fmt.Sprintf("%s (unavailable: %s)", candidate, rep.Reason))
			continue
		}
		return Selection{
			Requested: requested,
			Resolved:  candidate,
			Trust:     trust,
			Fallback:  true,
			Reason:    fmt.Sprintf("%s unavailable (%s); fallback to %s per policy", requested, requestedRep.Reason, candidate),
		}, nil
	}
	return Selection{}, fmt.Errorf("%w; fallback candidates rejected: %s", baseErr, strings.Join(rejected, "; "))
}

func unavailableError(mode Mode, rep CapabilityReport) error {
	msg := fmt.Sprintf("runtime mode %s is unavailable", mode)
	if rep.Reason != "" {
		msg += ": " + rep.Reason
	}
	if len(rep.Missing) > 0 {
		msg += fmt.Sprintf(" (missing: %s)", strings.Join(rep.Missing, ", "))
	}
	return fmt.Errorf("%s", msg)
}
