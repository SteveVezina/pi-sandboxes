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

// NewRouter creates the HTTP router with all endpoints and no server-side
// auth (Unix socket / loopback "local user trust" model).
func NewRouter(store *sandbox.Store, runStores ...*sandbox.AgentRunStore) *mux.Router {
	return newRouterWithAuth("", store, runStores...)
}

// newRouterWithAuth is NewRouter plus a server-side bearer-token gate. An
// empty authToken keeps the pass-through behavior (F23 T23.5, ADR-003).
func newRouterWithAuth(authToken string, store *sandbox.Store, runStores ...*sandbox.AgentRunStore) *mux.Router {
	runStore := sandbox.NewAgentRunStore()
	if len(runStores) > 0 && runStores[0] != nil {
		runStore = runStores[0]
	}

	router := mux.NewRouter()
	router.Use(guiCORSMiddleware)
	router.Use(bearerAuthMiddleware(authToken))
	router.Use(activeContextProxyMiddleware)
	router.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Health check
	router.HandleFunc("/health", HealthHandler()).Methods("GET")

	// Credentials (ADR-006 / F30 T30.7) — in-memory only
	router.HandleFunc("/v1/credentials", api.RegisterCredential).Methods("POST")
	router.HandleFunc("/v1/credentials", api.ListCredentials).Methods("GET")

	// Templates (F28 T28.1 / T28.2)
	router.HandleFunc("/v1/templates", api.ListTemplates).Methods("GET")
	router.HandleFunc("/v1/templates/fork", api.ForkTemplate).Methods("POST")
	router.HandleFunc("/v1/templates/validate", api.ValidateTemplate).Methods("POST")
	router.HandleFunc("/v1/templates/diff", api.DiffTemplates).Methods("POST")
	router.HandleFunc("/v1/templates/export", api.ExportTemplate).Methods("POST")
	router.HandleFunc("/v1/templates/import", api.ImportTemplate).Methods("POST")
	router.HandleFunc("/v1/templates/{name}/history", api.TemplateHistory).Methods("GET")
	router.HandleFunc("/v1/templates/{name}/rollback", api.RollbackTemplate).Methods("POST")
	router.HandleFunc("/v1/templates/{name}/promote", api.PromoteTemplate).Methods("POST")
	router.HandleFunc("/v1/templates/{name}", api.GetTemplate).Methods("GET")

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
	router.HandleFunc("/v1/sandboxes/{id}/files/list", api.FilesListSandbox(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/files/read", api.FilesReadSandbox(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}/files/write", api.FilesWriteSandbox(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/files/pull", api.FilesPullSandbox(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes/{id}/files/push", api.FilesPushSandbox(store)).Methods("POST")

	// Diff
	router.HandleFunc("/v1/sandboxes/{id}/diff", api.DiffSandbox(store)).Methods("GET")

	// Patch
	router.HandleFunc("/v1/sandboxes/{id}/patch", api.PatchSandbox(store)).Methods("GET")

	// Output (consolidated per PROP-009)
	router.HandleFunc("/v1/sandboxes/{id}/output", api.OutputSandbox(store)).Methods("GET", "POST")

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

	// Agent Run
	router.HandleFunc("/v1/sandboxes/{id}/agent-run", api.StartAgentRun(store, runStore)).Methods("POST")
	router.HandleFunc("/v1/agent-runs/{id}", api.GetAgentRun(runStore)).Methods("GET")
	router.HandleFunc("/v1/agent-runs/{id}/cancel", api.CancelAgentRun(runStore)).Methods("POST")
	router.HandleFunc("/v1/agent-runs", api.ListAgentRuns(runStore)).Methods("GET")

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
