package network

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// EgressDecision represents the result of an egress policy evaluation.
type EgressDecision int

const (
	EgressDeny EgressDecision = iota
	EgressAllow
	EgressAllowWithCredentials
)

// EgressResult represents an egress policy evaluation result.
type EgressResult struct {
	Decision EgressDecision
	Host     string
	Port     int
	Reason   string
	Creds    []string // credential IDs to inject
}

// EgressPolicy defines the policy for outbound traffic from sandboxes.
type EgressPolicy struct {
	mu            sync.RWMutex
	allowlist     []AllowlistEntry
	credentialMap map[string]CredentialRule
}

// AllowlistEntry defines a host/domain that is allowed for egress.
type AllowlistEntry struct {
	Host     string
	Port     int
	Protocol string // http, https, ssh, git
	Pattern  bool   // true for wildcard patterns
}

// CredentialRule defines how a credential is injected into outbound requests.
type CredentialRule struct {
	ID        string
	Name      string
	Type      string // git-token, registry-auth, etc.
	Hosts     []string
	Paths     []string
	InjectAs  string // header, env, file
	Redacted  bool   // redact from logs
}

// NewEgressPolicy creates a new egress policy.
func NewEgressPolicy() *EgressPolicy {
	return &EgressPolicy{
		allowlist:     make([]AllowlistEntry, 0),
		credentialMap: make(map[string]CredentialRule),
	}
}

// AddAllowlistEntry adds a host/domain to the allowlist.
func (p *EgressPolicy) AddAllowlistEntry(entry AllowlistEntry) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.allowlist = append(p.allowlist, entry)
}

// AddCredentialRule adds a credential injection rule.
func (p *EgressPolicy) AddCredentialRule(rule CredentialRule) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.credentialMap[rule.ID] = rule
}

// Evaluate evaluates an outbound request against the policy.
func (p *EgressPolicy) Evaluate(host string, port int, protocol string) *EgressResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if host is in the allowlist
	for _, entry := range p.allowlist {
		if matchesHost(entry.Host, host) && (entry.Port == 0 || entry.Port == port) {
			// Find matching credential rules
			var creds []string
			for _, rule := range p.credentialMap {
				if matchesHost(rule.Hosts[0], host) {
					creds = append(creds, rule.ID)
				}
			}

			if len(creds) > 0 {
				return &EgressResult{
					Decision: EgressAllowWithCredentials,
					Host:     host,
					Port:     port,
					Reason:   "allowlisted with credential injection",
					Creds:    creds,
				}
			}
			return &EgressResult{
				Decision: EgressAllow,
				Host:     host,
				Port:     port,
				Reason:   "allowlisted",
			}
		}
	}

	return &EgressResult{
		Decision: EgressDeny,
		Host:     host,
		Port:     port,
		Reason:   "not allowlisted",
	}
}

// matchesHost checks if a pattern matches a host.
func matchesHost(pattern, host string) bool {
	if pattern == host {
		return true
	}
	// Support wildcard patterns (*.example.com)
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return strings.HasSuffix(host, suffix)
	}
	return false
}

// ParseURL extracts host, port, and protocol from a URL.
func ParseURL(rawURL string) (host string, port int, protocol string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", 0, "", fmt.Errorf("parse URL: %w", err)
	}

	host = u.Hostname()
	protocol = u.Scheme

	if u.Port() != "" {
		port = 0 // Will be parsed by net/url
	} else {
		switch protocol {
		case "https":
			port = 443
		case "http":
			port = 80
		case "ssh":
			port = 22
		default:
			port = 0
		}
	}

	return host, port, protocol, nil
}
