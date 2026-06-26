package daemon

import (

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/api"
	"github.com/pi-sandbox/pi/pkg/session"
)

// NewRouter creates the HTTP router with all endpoints.
func NewRouter(store *session.Store) *mux.Router {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", HealthHandler()).Methods("GET")

	// Sandbox CRUD
	router.HandleFunc("/v1/sandboxes", api.CreateSandbox(store)).Methods("POST")
	router.HandleFunc("/v1/sandboxes", api.ListSandboxes(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}", api.GetSandbox(store)).Methods("GET")
	router.HandleFunc("/v1/sandboxes/{id}", api.DeleteSandbox(store)).Methods("DELETE")

	return router
}
