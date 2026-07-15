package daemon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pi-sandbox/pi/pkg/session"
)

// Daemon is the pi-sandboxd daemon.
type Daemon struct {
	socketPath string
	httpPort   int
	store      *session.Store
	server     *http.Server
}

// New creates a new daemon with the given session store.
func New(socketPath string, httpPort int, store *session.Store) *Daemon {
	return &Daemon{
		socketPath: socketPath,
		httpPort:   httpPort,
		store:      store,
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
	session.OrphanCleanup(d.store, d.store.Dir())

	// Create Unix socket listener
	unixListener, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("listen on unix socket: %w", err)
	}

	// Set socket permissions
	os.Chmod(d.socketPath, 0755)

	// Create HTTP server with router
	router := NewRouter(d.store)
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
