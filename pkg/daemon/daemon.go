package daemon

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// Daemon is the pi-sandboxd daemon.
type Daemon struct {
	socketPath string
	httpPort   int
	store      *sandbox.Store
	runStore   *sandbox.AgentRunStore
	server     *http.Server
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
	router := NewRouter(d.store, d.runStore)
	d.server = &http.Server{Handler: router}

	// Start HTTP server on Unix socket
	go func() {
		if err := d.server.Serve(unixListener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "daemon: serve error: %v\n", err)
		}
	}()

	// Start HTTP listener if port specified
	if d.httpPort > 0 {
		httpListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", d.httpPort))
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
