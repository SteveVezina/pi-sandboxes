package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pi-sandbox/pi/pkg/session"
)

// CreateRequest is the request body for sandbox creation.
type CreateRequest struct {
	Template  string `json:"template"`
	Mode      string `json:"mode"`
	Name      string `json:"name"`
	TTL       int    `json:"ttlSeconds"`
	Workspace struct {
		Mode    string `json:"mode"`
		Source  string `json:"source"`
		MaxSize string `json:"maxSize"`
	} `json:"workspace"`
}

// CreateSandbox returns an HTTP handler that creates a sandbox session.
func CreateSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}
		if req.Template == "" {
			req.Template = "base"
		}
		if req.Mode == "" {
			req.Mode = "fast"
		}
		workspaceMode := req.Workspace.Mode
		if workspaceMode == "" {
			workspaceMode = "copy"
		}
		if err := validateWorkspaceSource(workspaceMode, req.Workspace.Source); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		id, err := store.CreateWithOptions(session.CreateOptions{
			Name:          req.Name,
			Template:      req.Template,
			Mode:          req.Mode,
			TTL:           req.TTL,
			Workspace:     req.Workspace.Source,
			WorkspaceMode: workspaceMode,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Update state to warm
		store.UpdateState(id, session.StateWarm)

		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

func validateWorkspaceSource(mode, source string) error {
	if mode != "bind" {
		return nil
	}
	if source == "" {
		return fmt.Errorf("bind workspace mode requires a source")
	}

	clean := filepath.Clean(source)
	for _, blocked := range unsafeWorkspaceSources() {
		if blocked.recursive {
			if sameOrChild(clean, blocked.path) {
				return fmt.Errorf("unsafe workspace source %s", source)
			}
			continue
		}
		if clean == filepath.Clean(blocked.path) {
			return fmt.Errorf("unsafe workspace source %s", source)
		}
	}
	return nil
}

type unsafeWorkspaceSource struct {
	path      string
	recursive bool
}

func unsafeWorkspaceSources() []unsafeWorkspaceSource {
	var paths []unsafeWorkspaceSource
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		paths = append(paths,
			unsafeWorkspaceSource{path: home},
			unsafeWorkspaceSource{path: filepath.Join(home, ".ssh"), recursive: true},
			unsafeWorkspaceSource{path: filepath.Join(home, ".kube"), recursive: true},
			unsafeWorkspaceSource{path: filepath.Join(home, ".config", "gcloud"), recursive: true},
			unsafeWorkspaceSource{path: filepath.Join(home, ".config", "aws"), recursive: true},
		)
	}
	paths = append(paths,
		unsafeWorkspaceSource{path: "/var/run/docker.sock"},
		unsafeWorkspaceSource{path: "/run/docker.sock"},
	)
	return paths
}

func sameOrChild(path, parent string) bool {
	parent = filepath.Clean(parent)
	if path == parent {
		return true
	}
	rel, err := filepath.Rel(parent, path)
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !hasDotDotPrefix(rel)
}

func hasDotDotPrefix(path string) bool {
	return path == ".." || len(path) > 3 && path[:3] == "../"
}
