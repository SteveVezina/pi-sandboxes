package network

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pi-sandbox/pi/pkg/secrets"
)

// EgressProxy is a daemon-owned proxy that enforces egress policy and injects credentials.
type EgressProxy struct {
	policy    *EgressPolicy
	credStore *secrets.CredentialStore
	transport http.RoundTripper
}

// NewEgressProxy creates a new egress proxy.
func NewEgressProxy(policy *EgressPolicy, credStore *secrets.CredentialStore) *EgressProxy {
	return &EgressProxy{
		policy:    policy,
		credStore: credStore,
		transport: http.DefaultTransport,
	}
}

// RoundTrip implements http.RoundTripper to intercept and enforce egress policy.
func (p *EgressProxy) RoundTrip(req *http.Request) (*http.Response, error) {
	// Parse the request URL
	u, err := url.Parse(req.URL.String())
	if err != nil {
		return nil, fmt.Errorf("parse request URL: %w", err)
	}

	host := u.Hostname()
	port := 0
	if u.Port() != "" {
		port = 443 // Default for https
		if u.Scheme == "http" {
			port = 80
		}
	}

	// Evaluate policy
	result := p.policy.Evaluate(host, port, u.Scheme)

	switch result.Decision {
	case EgressDeny:
		return nil, fmt.Errorf("egress denied: %s (host: %s)", result.Reason, host)

	case EgressAllow:
		// Allow without credential injection
		return p.transport.RoundTrip(req)

	case EgressAllowWithCredentials:
		// Inject credentials
		req = p.injectCredentials(req, result.Creds)
		return p.transport.RoundTrip(req)

	default:
		return nil, fmt.Errorf("unknown egress decision")
	}
}

// injectCredentials injects credentials into the request.
func (p *EgressProxy) injectCredentials(req *http.Request, credIDs []string) *http.Request {
	for _, credID := range credIDs {
		cred, err := p.credStore.Get(credID)
		if err != nil {
			continue
		}

		switch cred.InjectAs {
		case "header":
			// Inject as HTTP header
			// In production, this would retrieve the actual credential value securely
			req.Header.Set(cred.Name, "[credential-injected]")

		case "env":
			// Env injection is not supported for HTTP requests
			// This would be handled at the process level

		case "file":
			// File injection is not supported for HTTP requests
			// This would be handled at the process level
		}
	}
	return req
}

// ProxyRoundTrip is a convenience function to make an HTTP request through the proxy.
func (p *EgressProxy) ProxyRoundTrip(ctx context.Context, method, rawURL string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return p.RoundTrip(req)
}

// IsAllowlisted checks if a host is allowlisted.
func (p *EgressProxy) IsAllowlisted(host string) bool {
	result := p.policy.Evaluate(host, 0, "")
	return result.Decision != EgressDeny
}

// GetDecision returns the egress decision for a host.
func (p *EgressProxy) GetDecision(host string) EgressDecision {
	result := p.policy.Evaluate(host, 0, "")
	return result.Decision
}

// LogDecision logs an egress decision (credentials are redacted).
func LogDecision(host string, decision EgressDecision, reason string) string {
	// Redact any sensitive information from the reason
	redactedReason := secrets.Redact(reason)
	return fmt.Sprintf("egress: %s -> %s (%s)", host, decisionLabel(decision), redactedReason)
}

func decisionLabel(d EgressDecision) string {
	switch d {
	case EgressDeny:
		return "DENY"
	case EgressAllow:
		return "ALLOW"
	case EgressAllowWithCredentials:
		return "ALLOW+CRED"
	default:
		return "UNKNOWN"
	}
}

// IsGitURL checks if a URL is a Git URL.
func IsGitURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "git@") ||
		strings.HasPrefix(rawURL, "https://") ||
		strings.HasPrefix(rawURL, "http://") ||
		strings.HasSuffix(rawURL, ".git")
}

// ParseGitHost extracts the host from a Git URL.
func ParseGitHost(rawURL string) (string, error) {
	if strings.HasPrefix(rawURL, "git@") {
		// git@github.com:path/repo.git
		parts := strings.Split(rawURL, "@")
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid git URL")
		}
		host := strings.Split(parts[1], ":")[0]
		return host, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}
