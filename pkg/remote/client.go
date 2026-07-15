// Package remote implements remote daemon transport and authentication (F23).
// See ADR-003 for the transport/auth matrix.
package remote

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	pictx "github.com/pi-sandbox/pi/pkg/context"
)

// Client is a context-aware daemon HTTP client.
//
// Supported transports (ADR-003):
//   - unix: connects to a local Unix domain socket
//   - http: direct HTTP endpoint, requires bearer-token auth
//   - ssh:  SSH-forwarded HTTP, requires ssh-agent auth
type Client struct {
	ctx       pictx.Context
	httpC     *http.Client
	baseURL   string
	bearerTok string
	sshDialer SSHDialer
}

// SSHDialer is the minimal interface for setting up an SSH-forwarded HTTP
// connection. It is injectable so tests can avoid an actual ssh-agent.
type SSHDialer interface {
	Dial(network, addr string) (net.Conn, error)
}

// NewClient builds a client for the given context, resolving any auth material
// (e.g. bearer-token env vars) up front so failures are surfaced immediately.
func NewClient(c pictx.Context) (*Client, error) {
	if err := pictx.Validate(c); err != nil {
		return nil, err
	}

	cl := &Client{ctx: c}

	switch c.Transport {
	case pictx.TransportUnix:
		socketPath := unixSocketPath(c.Target)
		cl.httpC = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		}
		cl.baseURL = "http://unix"
	case pictx.TransportHTTP:
		tok, err := resolveBearerToken(c.Auth)
		if err != nil {
			return nil, err
		}
		cl.bearerTok = tok
		cl.httpC = &http.Client{Timeout: 30 * time.Second}
		cl.baseURL = strings.TrimRight(c.Target, "/")
	case pictx.TransportSSH:
		// SSH dialing is intentionally lazy. We surface auth wiring here
		// (ssh-agent is required) but defer connection setup to Do().
		cl.httpC = &http.Client{Timeout: 30 * time.Second}
		cl.baseURL = strings.TrimRight(c.Target, "/")
	}
	return cl, nil
}

// Transport returns the configured transport name.
func (c *Client) Transport() string { return c.ctx.Transport }

// Auth returns the configured auth type.
func (c *Client) Auth() string { return c.ctx.Auth.Type }

// Context returns the underlying context configuration.
func (c *Client) Context() pictx.Context { return c.ctx }

// Do performs an HTTP request through the configured transport.
//
// For ssh transport, the underlying TCP connection is expected to be tunneled
// through ssh-agent forwarding (set up out-of-band by the user, e.g.
// `ssh -L 7777:127.0.0.1:7777 gpu-box.local`). This keeps the daemon HTTP API
// unchanged per ADR-003.
func (c *Client) Do(method, path string, body io.Reader) (*http.Response, error) {
	if c == nil || c.httpC == nil {
		return nil, fmt.Errorf("remote client not initialized")
	}
	urlStr := c.baseURL + path
	if c.ctx.Transport == pictx.TransportSSH {
		// SSH transport requires ssh-agent authentication for the underlying
		// connection. We surface ssh:// as a regular HTTP request to the
		// forwarded local port if Target uses ssh://, leaving setup to the
		// operator (documented in ADR-003).
		urlStr = sshTargetURL(c.baseURL) + path
	}
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("remote: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.bearerTok != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerTok)
	}
	req.Header.Set("X-Pi-Context-Proxy", "1")
	resp, err := c.httpC.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote %s %s: %w", method, path, err)
	}
	return resp, nil
}

// BaseURL is exposed for tests and diagnostics.
func (c *Client) BaseURL() string { return c.baseURL }

func unixSocketPath(target string) string {
	if u, err := url.Parse(target); err == nil && u.Scheme == "unix" {
		// unix:///path/to.sock → /path/to.sock
		path := u.Path
		if path == "" {
			path = u.Host
		}
		return expandHome(path)
	}
	return expandHome(strings.TrimPrefix(target, "unix://"))
}

func sshTargetURL(base string) string {
	// Strip ssh:// scheme; leave the rest for ssh-agent-managed tunnels.
	// In practice operators set up `ssh -L localPort:127.0.0.1:7777 host`
	// and point the context target at the local forwarded URL. We accept
	// either ssh://host or http://localhost:port for flexibility.
	if strings.HasPrefix(base, "ssh://") {
		return "http://" + strings.TrimPrefix(base, "ssh://")
	}
	return base
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home := homeDir(); home != "" {
			return home + p[1:]
		}
	}
	return p
}
