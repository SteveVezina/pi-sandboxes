package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// DeleteSandbox returns an HTTP handler that deletes a sandbox.
func DeleteSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Transition to destroying
		store.UpdateState(id, sandbox.StateDestroying)

		// Then delete metadata
		if err := store.Delete(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
	}
}
