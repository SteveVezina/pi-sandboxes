package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// DeleteSandbox returns an HTTP handler that deletes a sandbox.
func DeleteSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}

		// Transition to destroying
		store.UpdateState(id, sandbox.StateDestroying)

		containerName := compatContainerName(id)
		hasCompatContainer := meta.Mode == "compat"
		if exists, _ := compat.ContainerExists(containerName); exists {
			hasCompatContainer = true
		}
		if hasCompatContainer {
			c := &compat.Container{
				ID: id,
				Spec: &compat.ContainerSpec{
					ID:   id,
					Name: containerName,
				},
				Ready: true,
			}
			if err := c.Destroy(); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("destroy compat container: %v", err)})
				return
			}
			// Remove daemon-managed volumes (workspace, artifacts, caches).
			// Best-effort: a leaked volume must not block sandbox destruction.
			if err := compat.RemoveManagedVolumes(id); err != nil {
				log.Printf("sandbox: remove managed volumes for %s: %v", id, err)
			}
		}

		// Then delete metadata
		if err := store.Delete(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
	}
}

func compatContainerName(id string) string {
	if len(id) <= 8 {
		return "pi-sandbox-" + id
	}
	return "pi-sandbox-" + id[:8]
}
