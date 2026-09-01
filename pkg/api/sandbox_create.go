package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pi-sandbox/pi/pkg/cache"
	"github.com/pi-sandbox/pi/pkg/events"
	"github.com/pi-sandbox/pi/pkg/network"
	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/compat"
	"github.com/pi-sandbox/pi/pkg/runtime/detect"
	"github.com/pi-sandbox/pi/pkg/sandbox"
	"github.com/pi-sandbox/pi/pkg/template"
	"gopkg.in/yaml.v3"
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
	Network struct {
		Mode  string   `json:"mode"`  // none|restricted|open; default restricted
		Allow []string `json:"allow"` // extra restricted-mode allowlist hosts
	} `json:"network"`
}

// CreateSandbox returns an HTTP handler that creates a sandbox.
func CreateSandbox(store *sandbox.Store) http.HandlerFunc {
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
			req.Template = templateStore().Default() // promoted default (F28 T28.2c)
		}
		if req.Template == "" {
			req.Template = "base"
		}
		requested := req.Mode
		if requested == "" {
			requested = string(pruntime.ModeAuto)
		}
		sel, err := pruntime.Select(r.Context(), detect.DefaultRegistry(""),
			pruntime.Mode(requested), pruntime.TrustTrusted, pruntime.FallbackPolicy{})
		if err != nil {
			status := http.StatusBadRequest
			if requested == string(pruntime.ModeAuto) {
				status = http.StatusServiceUnavailable
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		req.Mode = string(sel.Resolved)
		workspaceMode := req.Workspace.Mode
		if workspaceMode == "" {
			workspaceMode = "copy"
		}
		if err := validateWorkspaceSource(workspaceMode, req.Workspace.Source); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		netMode := req.Network.Mode
		if netMode == "" {
			netMode = "restricted" // ADR-006 default
		}
		if _, err := network.PolicyFor(netMode, req.Network.Allow); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		id, err := store.CreateWithOptions(sandbox.CreateOptions{
			Name:           req.Name,
			Template:       req.Template,
			Mode:           req.Mode,
			RequestedMode:  string(sel.Requested),
			FallbackReason: sel.Reason,
			NetworkMode:    netMode,
			NetworkAllow:   req.Network.Allow,
			TTL:            req.TTL,
			Workspace:      req.Workspace.Source,
			WorkspaceMode:  workspaceMode,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// For compat mode, resolve the template image and create the container
		if req.Mode == "compat" {
			if err := createCompatContainer(store, id, req.Template); err != nil {
				// Clean up the sandbox if container creation fails
				store.Delete(id)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("container creation failed: %v", err)})
				return
			}
		}

		// Update state to warm
		store.UpdateState(id, sandbox.StateWarm)

		events.Emit(events.Event{
			Type:      events.SandboxCreated,
			SandboxID: id,
			Data: map[string]any{
				"template": req.Template,
				"mode":     req.Mode,
				"network":  netMode,
			},
		})

		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

// createCompatContainer resolves the template image and creates a Docker container.
func createCompatContainer(store *sandbox.Store, sandboxID, templateName string) error {
	// Load the template
	tmplStore := template.NewStore(filepath.Join(os.Getenv("HOME"), ".pi-box", "templates"))
	t, err := tmplStore.Get(templateName)
	if err != nil {
		// Use default templates if template file doesn't exist
		defaults := template.DefaultTemplates()
		yamlData, ok := defaults[templateName]
		if !ok {
			return fmt.Errorf("template %s not found", templateName)
		}
		var defaultTemplate template.Template
		if err := yaml.Unmarshal([]byte(yamlData), &defaultTemplate); err != nil {
			return fmt.Errorf("parse default template %s: %w", templateName, err)
		}
		t = &defaultTemplate
	}

	// Resolve the image
	image := template.ResolveTemplateImage(t)

	meta, _ := store.Get(sandboxID)

	// Cache mounts are daemon-managed named volumes scoped by
	// template/runtime/user (ADR-009), so sibling sandboxes of the same
	// template share a warm cache. No host bind mounts.
	caches := make(map[string]string)
	for name := range t.Caches {
		caches[name] = cacheVolumeName(t.Name, metaMode(meta), name)
	}

	// Create the container
	spec := &compat.ContainerSpec{
		ID:        sandboxID,
		Image:     image,
		Workspace: managedVolumeName("workspace", sandboxID),
		Artifacts: compatArtifactsSource(sandboxID),
		Caches:    caches,
	}
	if meta != nil {
		spec.Network = sandboxEgressNetwork(meta)
	}

	container, err := compat.CreateContainer(spec)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	// Verify the container is running
	state := container.State()
	if state != "running" {
		container.Destroy()
		return fmt.Errorf("container not running after creation: %s", state)
	}

	// Fresh named volumes are mounted root-owned; hand them to the
	// sandbox user so workspace and artifact writes work.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := container.ExecAsRoot(ctx, "chown 1000:1000 /workspace /artifacts /cache/* 2>/dev/null || chown 1000:1000 /workspace /artifacts"); err != nil {
		container.Destroy()
		return fmt.Errorf("prepare volumes: %w", err)
	}

	return nil
}

func managedVolumeName(parts ...string) string {
	cleaned := make([]string, 0, len(parts)+1)
	cleaned = append(cleaned, "pi-sandbox")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var b strings.Builder
		for _, r := range part {
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
				b.WriteRune(r)
			case r == '.', r == '_', r == '-':
				b.WriteRune(r)
			default:
				b.WriteByte('-')
			}
		}
		cleaned = append(cleaned, b.String())
	}
	return strings.Join(cleaned, "-")
}

func compatWorkspaceSource(id string, meta *sandbox.Meta) string {
	if meta != nil && meta.WorkspaceMode == "bind" && strings.TrimSpace(meta.Workspace) != "" {
		return meta.Workspace
	}
	return managedVolumeName("workspace", id)
}

// cacheVolumeName is the daemon-managed named volume for a cache type,
// scoped by template/runtime/user (ADR-009) rather than sandbox ID.
func cacheVolumeName(template, runtime, cacheType string) string {
	return managedVolumeName("cache", cache.VolumeScope(template, runtime, "default"), cacheType)
}

func metaMode(m *sandbox.Meta) string {
	if m == nil || m.Mode == "" {
		return "auto"
	}
	return m.Mode
}

func compatArtifactsSource(id string) string {
	return managedVolumeName("artifacts", id)
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
