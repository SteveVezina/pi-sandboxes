package network

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

// PolicyResolver returns the egress policy for the sandbox that originated a
// proxied request. A non-nil error (unknown sandbox, missing credentials)
// causes the request to be denied.
type PolicyResolver func(sandboxID string) (*Policy, error)

// Decision is a single egress ruling, handed to a DecisionSink for logging.
// It never carries credential material.
type Decision struct {
	SandboxID string
	Host      string
	Allowed   bool
	Reason    string
}

// DecisionSink records egress decisions (T30.6 wires this to sandbox
// logs/history). A nil sink is replaced with a slog-based default.
type DecisionSink func(Decision)

// HeaderInjection is one request header the proxy adds to an approved
// outbound request on the sandbox's behalf (T30.8). The value never
// leaves the daemon except on the wire to the approved host.
type HeaderInjection struct {
	Name  string
	Value string
}

// CredentialInjector returns the headers to inject for an approved host,
// or nil. It is only consulted for plaintext HTTP forwarding — CONNECT
// tunnels are end-to-end encrypted and cannot be modified in the no-MITM
// baseline (ADR-006).
type CredentialInjector func(host string) []HeaderInjection

// ProxyServer is the daemon-owned forward proxy. Every sandbox in
// restricted mode routes outbound HTTP(S) through one ProxyServer instance;
// it enforces the per-sandbox allowlist, injects approved credentials, and
// records decisions.
type ProxyServer struct {
	resolve   PolicyResolver
	sink      DecisionSink
	inject    CredentialInjector
	transport http.RoundTripper
	dialer    *net.Dialer
}

// SetCredentialInjector wires credential injection for approved plaintext
// HTTP requests. Safe to leave unset (no injection).
func (p *ProxyServer) SetCredentialInjector(fn CredentialInjector) { p.inject = fn }

// NewProxyServer builds a proxy that resolves per-sandbox policy via resolve
// and reports decisions to sink (nil sink → slog default).
func NewProxyServer(resolve PolicyResolver, sink DecisionSink) *ProxyServer {
	if sink == nil {
		// Reason strings here are fixed internal labels, never credential
		// material; T30.6 adds the real sandbox-scoped sink.
		sink = func(d Decision) {
			slog.Info("egress decision",
				"sandbox_id", d.SandboxID, "host", d.Host,
				"allowed", d.Allowed, "reason", d.Reason)
		}
	}
	return &ProxyServer{
		resolve:   resolve,
		sink:      sink,
		transport: http.DefaultTransport,
		dialer:    &net.Dialer{Timeout: 15 * time.Second},
	}
}

// ServeHTTP implements the forward-proxy protocol: CONNECT for HTTPS
// tunnels, plain proxying for HTTP.
func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sandboxID, ok := sandboxIDFromProxyAuth(r)
	if !ok {
		http.Error(w, "proxy: missing sandbox identity", http.StatusProxyAuthRequired)
		return
	}

	target := r.Host
	if r.Method != http.MethodConnect && r.URL.Host != "" {
		target = r.URL.Host
	}
	host := hostOnly(target)

	policy, err := p.resolve(sandboxID)
	if err != nil || policy == nil {
		p.sink(Decision{sandboxID, host, false, "unknown sandbox or policy"})
		http.Error(w, "proxy: egress denied", http.StatusForbidden)
		return
	}

	if !policy.IsAllowed(host) {
		p.sink(Decision{sandboxID, host, false, "not allowlisted"})
		http.Error(w, "proxy: egress denied for "+host, http.StatusForbidden)
		return
	}
	p.sink(Decision{sandboxID, host, true, "allowlisted"})

	if r.Method == http.MethodConnect {
		p.tunnel(w, target)
		return
	}
	p.forward(w, r)
}

func (p *ProxyServer) tunnel(w http.ResponseWriter, target string) {
	if _, _, err := net.SplitHostPort(target); err != nil {
		target = net.JoinHostPort(target, "443")
	}
	upstream, err := p.dialer.Dial("tcp", target)
	if err != nil {
		http.Error(w, "proxy: upstream dial failed", http.StatusBadGateway)
		return
	}
	defer upstream.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "proxy: hijack unsupported", http.StatusInternalServerError)
		return
	}
	client, _, err := hj.Hijack()
	if err != nil {
		return
	}
	defer client.Close()

	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return
	}

	done := make(chan struct{}, 2)
	go func() { io.Copy(upstream, client); done <- struct{}{} }()
	go func() { io.Copy(client, upstream); done <- struct{}{} }()
	<-done
}

func (p *ProxyServer) forward(w http.ResponseWriter, r *http.Request) {
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	outReq.Header.Del("Proxy-Authorization")
	outReq.Header.Del("Proxy-Connection")

	if p.inject != nil {
		if headers := p.inject(hostOnly(outReq.Host)); len(headers) > 0 {
			for _, h := range headers {
				// The proxy owns the injected header — a sandbox cannot
				// smuggle its own value past it.
				outReq.Header.Set(h.Name, h.Value)
			}
		}
	}

	resp, err := p.transport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, "proxy: upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// sandboxIDFromProxyAuth reads the sandbox ID from the basic-auth username
// of the Proxy-Authorization header (T30.5 injects it as HTTP_PROXY creds).
func sandboxIDFromProxyAuth(r *http.Request) (string, bool) {
	const prefix = "Basic "
	h := r.Header.Get("Proxy-Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	raw, err := base64.StdEncoding.DecodeString(h[len(prefix):])
	if err != nil {
		return "", false
	}
	user, _, found := strings.Cut(string(raw), ":")
	if !found || user == "" {
		return "", false
	}
	return user, true
}

func hostOnly(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}
