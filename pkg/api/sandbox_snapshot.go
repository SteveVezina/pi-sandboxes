package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
	"github.com/pi-sandbox/pi/pkg/snapshot"
)

// snapshotFromContainer copies /workspace out of the sandbox container and
// stores it as a named snapshot in the daemon-owned snapshot store.
func snapshotFromContainer(r *http.Request, id, name string) error {
	c, err := compatContainerHandle(id)
	if err != nil {
		return err
	}
	staging, err := os.MkdirTemp("", "pi-box-snapshot-*")
	if err != nil {
		return fmt.Errorf("stage snapshot: %w", err)
	}
	defer os.RemoveAll(staging)

	if err := c.CopyFrom(workspaceRoot+"/.", staging); err != nil {
		return fmt.Errorf("copy workspace: %w", err)
	}
	sm := snapshot.NewManager(id)
	if _, err := sm.Create(name, staging); err != nil {
		return err
	}
	return nil
}

// restoreToContainer replaces /workspace inside the sandbox container with
// the content of a named snapshot.
func restoreToContainer(r *http.Request, id, name string) error {
	c, err := compatContainerHandle(id)
	if err != nil {
		return err
	}
	staging, err := os.MkdirTemp("", "pi-box-rollback-*")
	if err != nil {
		return fmt.Errorf("stage rollback: %w", err)
	}
	defer os.RemoveAll(staging)

	sm := snapshot.NewManager(id)
	if err := sm.Rollback(name, staging); err != nil {
		return err
	}

	// Clear the live workspace, then copy the snapshot content back in.
	if _, err := workspaceExec(r.Context(), id,
		"find "+workspaceRoot+" -mindepth 1 -maxdepth 1 -exec rm -rf {} +"); err != nil {
		return fmt.Errorf("clear workspace: %w", err)
	}
	if err := c.CopyTo(staging+"/.", workspaceRoot); err != nil {
		return fmt.Errorf("restore workspace: %w", err)
	}
	// docker cp writes as root; hand the files back to the sandbox user.
	if err := c.ExecAsRoot(r.Context(), "chown -R 1000:1000 "+workspaceRoot); err != nil {
		return fmt.Errorf("fix ownership: %w", err)
	}
	return nil
}

// SnapshotSandbox returns an HTTP handler for snapshot operations.
func SnapshotSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if err := requireCompat(meta); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		// Parse request
		var req struct {
			Action string `json:"action"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Name == "" && req.Action != "list" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		switch req.Action {
		case "create", "":
			if err := snapshotFromContainer(r, id, req.Name); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create snapshot: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":     id,
				"action": "create",
				"name":   req.Name,
			})

		case "list":
			sm := snapshot.NewManager(id)
			list, err := sm.List()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list snapshots: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":        id,
				"action":    "list",
				"snapshots": list,
			})

		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action: " + req.Action})
		}
	}
}

// SnapshotCreate returns an HTTP handler that creates a snapshot.
func SnapshotCreate(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if err := requireCompat(meta); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		if err := snapshotFromContainer(r, id, req.Name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create snapshot: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":     id,
			"action": "create",
			"name":   req.Name,
		})
	}
}

// SnapshotList returns an HTTP handler that lists snapshots.
func SnapshotList(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		sm := snapshot.NewManager(id)
		list, err := sm.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list snapshots: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":        id,
			"action":    "list",
			"snapshots": list,
		})
	}
}

// SnapshotRollback returns an HTTP handler that rolls back to a snapshot.
func SnapshotRollback(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		meta, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if err := requireCompat(meta); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		if err := restoreToContainer(r, id, req.Name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "rollback: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":     id,
			"action": "rollback",
			"name":   req.Name,
		})
	}
}

// SnapshotDelete returns an HTTP handler that deletes a snapshot.
func SnapshotDelete(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		sm := snapshot.NewManager(id)
		if err := sm.Delete(req.Name); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete snapshot: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":     id,
			"action": "delete",
			"name":   req.Name,
		})
	}
}
