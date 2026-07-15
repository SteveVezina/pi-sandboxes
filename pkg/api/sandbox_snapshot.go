package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/snapshot"
	"github.com/pi-sandbox/pi/pkg/workspace"
)

// SnapshotSandbox returns an HTTP handler for snapshot operations.
func SnapshotSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
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
		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		// Create workspace manager
		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		sm := snapshot.NewManager(id)

		switch req.Action {
		case "create", "":
			// Create snapshot
			if err := createSnapshot(id, mgr, sm, req.Name); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create snapshot: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":     id,
				"action": "create",
				"name":   req.Name,
			})

		case "list":
			// List snapshots
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
func SnapshotCreate(store *session.Store) http.HandlerFunc {
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

		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		sm := snapshot.NewManager(id)
		if err := createSnapshot(id, mgr, sm, req.Name); err != nil {
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
func SnapshotList(store *session.Store) http.HandlerFunc {
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
func SnapshotRollback(store *session.Store) http.HandlerFunc {
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

		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		sm := snapshot.NewManager(id)
		if err := sm.Rollback(req.Name, mgr.Dir()); err != nil {
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
func SnapshotDelete(store *session.Store) http.HandlerFunc {
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

// createSnapshot creates a snapshot by copying the workspace.
func createSnapshot(sandboxID string, mgr *workspace.Manager, sm *snapshot.Manager, name string) error {
	_, err := sm.Create(name, mgr.Dir())
	return err
}

// dirSize calculates the total size of a directory.
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
