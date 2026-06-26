package daemon

import (
	"github.com/gorilla/mux"
)

// NewRouter creates the HTTP router with all endpoints.
func NewRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/health", HealthHandler()).Methods("GET")
	return router
}
