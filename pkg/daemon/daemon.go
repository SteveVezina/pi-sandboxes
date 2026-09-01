package daemon

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/logs"
	"github.com/pi-sandbox/pi/pkg/network"
	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// Daemon is the pi-sandboxd daemon.
type Daemon struct {
	socketPath  string
	httpPort    int
	httpHost    string
	authToken   string
	proxyPort   int
	store       *sandbox.Store
	runStore    *sandbox.AgentRunStore
	server      *http.Server
	proxyServer *http.Server
}

// SetEgressProxyPort enables the daemon-owned egress proxy on
// 127.0.0.1:<port> (ADR-006). Zero (the default) leaves it disabled.
// Must be called before Start.
func (d *Daemon) SetEgressProxyPort(port int) {
	d.proxyPort = port
}

// SetHTTPHost sets the bind host for the optional HTTP listener. Empty means
// the default "127.0.0.1". Must be called before Start.
func (d *Daemon) SetHTTPHost(host string) {
	d.httpHost = host
}

// SetAuthToken sets the expected bearer token for server-side auth (F23
// T23.5). Empty disables the gate. Must be called before Start.
func (d *Daemon) SetAuthToken(token string) {
	d.authToken = token
}

// ProxyAddr returns the egress proxy address for injection into sandbox
// HTTP_PROXY env (T30.5), or "" when the proxy is disabled.
func (d *Daemon) ProxyAddr() string {
	if d.proxyPort <= 0 {
		return ""
	}
	return fmt.Sprintf("127.0.0.1:%d", d.proxyPort)
}

// egressDecisionSink routes egress-proxy decisions: denials are recorded
// in the sandbox's egress log (F30 T30.6) and logged at info; allows stay
// at debug in the daemon log only.
func (d *Daemon) egressDecisionSink(dec network.Decision) {
	if dec.Allowed {
		slog.Debug("egress allowed", "sandbox_id", dec.SandboxID, "host", dec.Host)
		return
	}
	slog.Info("egress denied", "sandbox_id", dec.SandboxID, "host", dec.Host, "reason", dec.Reason)
	if err := logs.NewManager(dec.SandboxID).RecordEgress(dec.Host, false, dec.Reason); err != nil {
		slog.Warn("record egress denial", "sandbox_id", dec.SandboxID, "err", err)
	}
}

// egressPolicyResolver rebuilds a sandbox's egress policy from its
// persisted network mode/allowlist. Sandboxes predating ADR-006 default to
// restricted.
func (d *Daemon) egressPolicyResolver(sandboxID string) (*network.Policy, error) {
	m, err := d.store.Get(sandboxID)
	if err != nil {
		return nil, err
	}
	mode := m.NetworkMode
	if mode == "" {
		mode = string(network.ModeRestricted)
	}
	return network.PolicyFor(mode, m.NetworkAllow)
}

// New creates a new daemon with the given store and optional agent run store.
func New(socketPath string, httpPort int, store *sandbox.Store, runStores ...*sandbox.AgentRunStore) *Daemon {
	runStore := sandbox.NewAgentRunStore()
	if len(runStores) > 0 && runStores[0] != nil {
		runStore = runStores[0]
	}

	return &Daemon{
		socketPath: socketPath,
		httpPort:   httpPort,
		store:      store,
		runStore:   runStore,
	}
}

// Start starts the daemon.
func (d *Daemon) Start() error {
	// Fail closed: a non-loopback HTTP bind must carry a bearer token
	// (F23 T23.5). The Unix socket and loopback HTTP keep local-user trust.
	host := d.httpHost
	if host == "" {
		host = "127.0.0.1"
	}
	if d.httpPort > 0 && !isLoopbackHost(host) && d.authToken == "" {
		return fmt.Errorf("refusing to start: --http-addr %s is not loopback and PI_DAEMON_TOKEN is unset "+
			"(a public daemon without auth is an unauthenticated sandbox-exec API)", host)
	}

	// Remove stale socket file
	if _, err := os.Stat(d.socketPath); err == nil {
		os.Remove(d.socketPath)
	}

	// Create parent directory if missing
	dir := filepath.Dir(d.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}

	// Run orphan cleanup on startup (PROP-008 D7: reconciliation)
	sandbox.OrphanCleanup(d.store, d.store.Dir())

	// Reconcile compat-mode containers against the sandbox store
	// (PROP-008 T15.2c): drop containers with no active sandbox, and
	// mark sandboxes DESTROYED if their container vanished out-of-band.
	d.reconcileCompatContainers()

	// Create Unix socket listener
	unixListener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("listen on unix socket: %w", err)
	}

	// Set socket permissions
	os.Chmod(d.socketPath, 0755)

	// Create HTTP server with router
	router := newRouterWithAuth(d.authToken, d.store, d.runStore)
	d.server = &http.Server{Handler: router}

	// Start HTTP server on Unix socket
	go func() {
		if err := d.server.Serve(unixListener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "daemon: serve error: %v\n", err)
		}
	}()

	// Start the daemon-owned egress proxy if enabled (ADR-006 / F30 T30.2)
	if d.proxyPort > 0 {
		proxyListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", d.proxyPort))
		if err != nil {
			fmt.Fprintf(os.Stderr, "daemon: warning: could not start egress proxy: %v\n", err)
		} else {
			proxy := network.NewProxyServer(d.egressPolicyResolver, d.egressDecisionSink)
			proxy.SetCredentialInjector(network.CredentialInjectorFromStore(api.CredentialStoreInstance()))
			d.proxyServer = &http.Server{Handler: proxy}
			api.SetEgressProxyAddr(d.ProxyAddr())
			go func() {
				if err := d.proxyServer.Serve(proxyListener); err != nil && err != http.ErrServerClosed {
					fmt.Fprintf(os.Stderr, "daemon: egress proxy serve error: %v\n", err)
				}
			}()
		}
	}

	// Start HTTP listener if port specified
	if d.httpPort > 0 {
		httpListener, err := net.Listen("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", d.httpPort)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "daemon: warning: could not start HTTP listener: %v\n", err)
		} else {
			go func() {
				if err := d.server.Serve(httpListener); err != nil && err != http.ErrServerClosed {
					fmt.Fprintf(os.Stderr, "daemon: HTTP serve error: %v\n", err)
				}
			}()
		}
	}

	return nil
}

// reconcileCompatContainers reconciles live OCI containers against
// WARM/EXECUTING compat-mode sandboxes in the store (PROP-008 T15.2c).
func (d *Daemon) reconcileCompatContainers() {
	ids, err := d.store.List()
	if err != nil {
		log.Printf("daemon: reconcile: list sandboxes: %v", err)
		return
	}

	var activeIDs []string
	for _, id := range ids {
		meta, err := d.store.Get(id)
		if err != nil {
			continue
		}
		if meta.Mode != string(pruntime.ModeCompat) {
			continue
		}
		if meta.State != sandbox.StateWarm && meta.State != sandbox.StateExecuting {
			continue
		}
		activeIDs = append(activeIDs, id)
	}

	if len(activeIDs) == 0 {
		return
	}

	result, err := compat.Reconcile(context.Background(), activeIDs)
	if err != nil {
		log.Printf("daemon: reconcile: compat containers: %v", err)
		return
	}

	for _, name := range result.RemovedContainers {
		log.Printf("daemon: reconcile: removed orphaned container %s", name)
	}
	for _, id := range result.MissingSandboxIDs {
		log.Printf("daemon: reconcile: sandbox %s container missing, marking DESTROYED", id)
		if err := d.store.UpdateState(id, sandbox.StateDestroyed); err != nil {
			log.Printf("daemon: reconcile: update state for %s: %v", id, err)
		}
	}
}

// Stop gracefully shuts down the daemon.
func (d *Daemon) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if d.proxyServer != nil {
		_ = d.proxyServer.Shutdown(ctx)
	}
	return d.server.Shutdown(ctx)
}

// HealthHandler returns the health check HTTP handler.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}
