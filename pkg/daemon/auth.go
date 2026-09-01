package daemon

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
)

// bearerAuthMiddleware enforces server-side bearer-token auth (F23 T23.5,
// ADR-003). When token is empty the middleware is a pass-through: the Unix
// socket and a loopback-only HTTP listener keep the "local user trust" model.
// When token is set, every request except GET /health and CORS pre-flight
// (OPTIONS) must carry "Authorization: Bearer <token>"; a missing or wrong
// token returns 401 and the wrapped handler is never called.
func bearerAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" || authExempt(r) {
				next.ServeHTTP(w, r)
				return
			}
			if !validBearer(r.Header.Get("Authorization"), token) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="pi-sandboxd"`)
				http.Error(w, `{"error":"unauthorized: valid bearer token required"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authExempt lists the requests that never require a token: the health probe
// (so load balancers / Render can reach it) and CORS pre-flight.
func authExempt(r *http.Request) bool {
	if r.Method == http.MethodOptions {
		return true
	}
	return r.Method == http.MethodGet && r.URL.Path == "/health"
}

func validBearer(header, want string) bool {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	got := strings.TrimSpace(header[len(prefix):])
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// isLoopbackHost reports whether host resolves to loopback only. An empty host
// or "localhost" counts as loopback; a literal like "0.0.0.0" or a routable IP
// does not.
func isLoopbackHost(host string) bool {
	switch host {
	case "", "localhost":
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
