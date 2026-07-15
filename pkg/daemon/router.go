package daemon

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/session"
)

// NewRouter creates the HTTP router with all endpoints.
func NewRouter(store *session.Store) *mux.Router {
	router := mux.NewRouter()
	router.Use(guiCORSMiddleware)
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
	router.HandleFunc("/v1/contexts/use", api.ContextUse()).Methods("POST")

	return router
}

func guiCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowedGUIOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
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
