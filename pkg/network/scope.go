package network

import (
	"fmt"
	"strings"
)

// ValidMode reports whether s is a recognized network mode.
func ValidMode(s string) bool {
	switch Mode(s) {
	case ModeNone, ModeRestricted, ModeOpen:
		return true
	default:
		return false
	}
}

// PolicyFor builds the per-sandbox network policy for a mode and optional
// extra allowlist hosts.
//
// DefaultDeny is applied in every mode, including open (ADR-006 §4 —
// overrides cannot relax default deny). extraAllow only widens the
// restricted-mode allowlist; it is ignored for none and open.
func PolicyFor(mode string, extraAllow []string) (*Policy, error) {
	if !ValidMode(mode) {
		return nil, fmt.Errorf("invalid network mode: %q (want none|restricted|open)", mode)
	}

	p := &Policy{Mode: Mode(mode), DenyList: DefaultDeny}

	if Mode(mode) == ModeRestricted {
		allow := make(DomainList, 0, len(DefaultAllowlist)+len(extraAllow))
		allow = append(allow, DefaultAllowlist...)
		for _, h := range extraAllow {
			if h = strings.TrimSpace(h); h != "" {
				allow = append(allow, h)
			}
		}
		p.Allowlist = allow
	}

	return p, nil
}
