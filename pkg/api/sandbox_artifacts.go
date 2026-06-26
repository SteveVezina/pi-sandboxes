package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/artifacts"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/workspace"
)

// ArtifactsSandbox returns an HTTP handler for artifact operations.
func ArtifactsSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		// Create workspace manager
		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		// Create artifact manager
		am := artifacts.NewManager(mgr.Dir())

		switch r.Method {
		case "GET":
			// List artifacts
			list, err := am.List()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list artifacts: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":    id,
				"files": list,
			})

		case "POST":
			// Check for export action
			var req struct {
				Action    string `json:"action"`
				Output    string `json:"output"`
				Destination string `json:"destination"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				// Default to pull
				req.Action = "pull"
			}

			switch req.Action {
			case "list":
				list, err := am.List()
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list artifacts: " + err.Error()})
					return
				}
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"id":    id,
					"files": list,
				})

			case "pull":
				dest := req.Destination
				if dest == "" {
					dest = "/tmp"
				}
				if err := am.Pull(dest); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pull artifacts: " + err.Error()})
					return
				}
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"id":        id,
					"action":    "pull",
					"destination": dest,
				})

			case "pack":
				output := req.Output
				if output == "" {
					output = "/tmp/artifacts.tar.gz"
				}
				if err := am.Pack(output); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pack artifacts: " + err.Error()})
					return
				}
				info, _ := os.Stat(output)
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"id":     id,
					"action": "pack",
					"output": output,
					"bytes":  info.Size(),
				})

			case "export":
				// Alias for pull
				dest := req.Destination
				if dest == "" {
					dest = "/tmp"
				}
				if err := am.Pull(dest); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "export artifacts: " + err.Error()})
					return
				}
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"id":        id,
					"action":    "export",
					"destination": dest,
				})

			default:
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action: " + req.Action})
			}

		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	}
}

// ArtifactsList returns an HTTP handler that lists artifacts for a sandbox.
func ArtifactsList(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		am := artifacts.NewManager(mgr.Dir())
		list, err := am.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list artifacts: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":    id,
			"files": list,
		})
	}
}

// ArtifactsPull returns an HTTP handler that pulls artifacts to a host destination.
func ArtifactsPull(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		am := artifacts.NewManager(mgr.Dir())

		// Get destination from query or body
		dest := r.URL.Query().Get("destination")
		if dest == "" {
			var req struct {
				Destination string `json:"destination"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				dest = req.Destination
			}
		}
		if dest == "" {
			dest = "/tmp"
		}

		if err := am.Pull(dest); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pull artifacts: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          id,
			"action":      "pull",
			"destination": dest,
		})
	}
}

// ArtifactsPack returns an HTTP handler that packs artifacts into an archive.
func ArtifactsPack(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		mgr := workspace.NewManager(id, workspace.ModeCopy)
		if err := mgr.EnsureDir(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ensure workspace: " + err.Error()})
			return
		}

		am := artifacts.NewManager(mgr.Dir())

		// Get output path from query or body
		output := r.URL.Query().Get("output")
		if output == "" {
			var req struct {
				Output string `json:"output"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				output = req.Output
			}
		}
		if output == "" {
			output = filepath.Join("/tmp", "artifacts.tar.gz")
		}

		if err := am.Pack(output); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "pack artifacts: " + err.Error()})
			return
		}

		info, _ := os.Stat(output)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":     id,
			"action": "pack",
			"output": output,
			"bytes":  info.Size(),
		})
	}
}
