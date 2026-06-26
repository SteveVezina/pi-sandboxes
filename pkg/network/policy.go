package network

import (
	"fmt"
	"strings"
)

// Policy defines network access rules for a sandbox session.
type Policy struct {
	Mode       Mode       `json:"mode"`
	Allowlist  DomainList `json:"allowlist"`
	DenyList   DomainList `json:"denylist"`
}

// DefaultPolicy returns the default network policy.
func DefaultPolicy() *Policy {
	return &Policy{
		Mode:      ModeRestricted,
		Allowlist: DefaultAllowlist,
		DenyList:  DefaultDeny,
	}
}

// Validate checks if the policy is valid.
func (p *Policy) Validate() error {
	switch p.Mode {
	case ModeNone, ModeRestricted, ModeOpen:
		return nil
	default:
		return fmt.Errorf("invalid network mode: %s", p.Mode)
	}
}

// IsAllowed checks if a host is allowed by this policy.
func (p *Policy) IsAllowed(host string) bool {
	// Check default deny first
	for _, denied := range p.DenyList {
		if denied == host || strings.HasSuffix(host, "."+denied) {
			return false
		}
	}

	switch p.Mode {
	case ModeNone:
		return false
	case ModeOpen:
		return true
	case ModeRestricted:
		if len(p.Allowlist) > 0 {
			return p.Allowlist.Contains(host)
		}
		// Fall back to default allowlist
		return DefaultAllowlist.Contains(host)
	default:
		return false
	}
}

// ApplyNetworkMode returns a new policy with the given mode.
func (p *Policy) ApplyNetworkMode(mode Mode) *Policy {
	result := *p
	result.Mode = mode
	return &result
}
