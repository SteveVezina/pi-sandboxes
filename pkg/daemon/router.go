package daemon

import (
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	pictx "github.com/pi-sandbox/pi/pkg/context"
	"github.com/pi-sandbox/pi/pkg/remote"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// NewRouter creates the HTTP router with all endpoints.
func NewRouter(store *session.Store) *mux.Router {
	router := mux.NewRouter()
	router.Use(guiCORSMiddleware)
	router.Use(activeContextProxyMiddleware)
	router.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Health check
	router.HandleFunc("/health", HealthHandler()).Methods("GET")

	// Sandbox CRUD
	router.HandleFunc("/v1/sandboxes", api.CreateSandbox(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes", api.ListSandboxes(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}", api.GetSandbox(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}", api.DeleteSandbox(store)).Methods("DELETE")

	// Exec
	router.HandleFunc("/v1/sandboxes/{id}/exec", api.ExecSandbox(store)).Methods("POST")

	// Interactive shell (WebSocket upgrade)
	router.HandleFunc("/v1/sandboxes/{id}/shell", api.ShellSandbox(store)).Methods("GET")

	// Clone
	router.HandleFunc("/v1/sandboxes/{id}/clone", api.CloneSandbox(store)).Methods("POST")

	// Files
	router.HandleFunc("/v1/sandboxes/{id}/files/read", api.FilesReadSandbox(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/files/write", api.FilesWriteSandbox(store)).Methods("POST")

	// Diff
	router.HandleFunc("/v1/sandboxes/{id}/diff", api.DiffSandbox(store)).Methods("GET")

	// Patch
	router.HandleFunc("/v1/sandboxes/{id}/patch", api.PatchSandbox(store)).Methods("GET")

	// Artifacts
	router.HandleFunc("/v1/sandboxes/{id}/artifacts", api.ArtifactsSandbox(store)).Methods("GET", "POST")
	router.HandleFunc("/v1/sandboxes/{id}/artifacts/list", api.ArtifactsList(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/artifacts/pull", api.ArtifactsPull(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/artifacts/pack", api.ArtifactsPack(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/artifacts/export", api.ArtifactsSandbox(store)).Methods("POST")

	// Snapshots
	router.HandleFunc("/v1/sandboxes/{id}/snapshot", api.SnapshotSandbox(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/snapshot/create", api.SnapshotCreate(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/snapshot/list", api.SnapshotList(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/snapshot/rollback", api.SnapshotRollback(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/snapshot/delete", api.SnapshotDelete(store)).Methods("POST")
	// SPEC.md §9 canonical rollback route (alias for backwards-compatibility with spec).
	router.HandleFunc("/v1/sandboxes/{id}/rollback", api.SnapshotRollback(store)).Methods("POST")

	// Logs
	router.HandleFunc("/v1/sandboxes/{id}/logs", api.LogsSandbox(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/logs/list", api.LogsList(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/logs/history", api.LogsHistory(store)).Methods("GET")

	// System diagnostics for GUI and integrations
	router.HandleFunc("/v1/system/status", api.SystemStatus(store)).Methods("GET")
	router.HandleFunc("/v1/system/doctor", api.SystemDoctor()).Methods("GET")
	router.HandleFunc("/v1/system/runtimes", api.SystemRuntimes()).Methods("GET")
	router.HandleFunc("/v1/support-bundle", api.SupportBundle(store)).Methods("GET")

	// Context selection for GUI and integrations
	router.HandleFunc("/v1/contexts", api.ContextsList()).Methods("GET")
	router.HandleFunc("/v1/contexts", api.ContextCreate()).Methods("POST")
	router.HandleFunc("/v1/contexts/{name}", api.ContextGet()).Methods("GET")
	router.HandleFunc("/v1/contexts/{name}", api.ContextUpdate()).Methods("PUT")
	router.HandleFunc("/v1/contexts/{name}", api.ContextDelete()).Methods("DELETE")
	router.HandleFunc("/v1/contexts/use", api.ContextUse()).Methods("POST")

	return router
}

func activeContextProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !shouldProxyActiveContext(r) {
			next.ServeHTTP(w, r)
			return
		}

		store, err := pictx.NewStore(pictx.DefaultPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		active, err := store.Active()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if active.Name == pictx.LocalContextName || active.Transport == pictx.TransportUnix {
			next.ServeHTTP(w, r)
			return
		}

		client, err := remote.NewClient(active)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		resp, err := client.Do(r.Method, r.URL.RequestURI(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		copyProxyHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})
}

func shouldProxyActiveContext(r *http.Request) bool {
	if r.Method == http.MethodOptions || r.Header.Get("X-Pi-Context-Proxy") != "" {
		return false
	}
	path := r.URL.Path
	if path == "/health" || strings.HasPrefix(path, "/v1/contexts") {
		return false
	}
	return strings.HasPrefix(path, "/v1/sandboxes") ||
		strings.HasPrefix(path, "/v1/system/") ||
		path == "/v1/support-bundle"
}

func copyProxyHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func isHopByHopHeader(key string) bool {
	switch strings.ToLower(key) {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
		"te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func guiCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedGUIOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Accept,Authorization")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedGUIOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	// Accept any localhost/127.0.0.1 origin (Vite dev server may use any port).
	if strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:") {
		return true
	}
	return false
}
