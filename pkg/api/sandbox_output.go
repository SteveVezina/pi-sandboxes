package api

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// tarGzDir archives the contents of dir into a .tar.gz file at output.
func tarGzDir(dir, output string) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "." {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(tw, src)
		return err
	})
}

// outputSources are the deliverable locations inside a sandbox (SPEC.md).
var outputSources = []string{
	"/artifacts",
	"/workspace/dist",
	"/workspace/build",
	"/workspace/coverage",
	"/workspace/test-results",
	"/workspace/target/release",
}

// OutputRequest is the request body for the output endpoint.
type OutputRequest struct {
	Action string `json:"action"` // "list", "pull", "pack"
	Dest   string `json:"dest,omitempty"`
	Output string `json:"output,omitempty"`
}

// OutputItem represents a deliverable output item.
type OutputItem struct {
	Path     string `json:"path"`
	Type     string `json:"type"` // "file", "directory", "patch"
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// OutputListResponse is the response for listing outputs.
type OutputListResponse struct {
	SandboxID string       `json:"sandbox_id"`
	Items     []OutputItem `json:"items"`
}

// OutputPullResponse is the response for pulling outputs.
type OutputPullResponse struct {
	SandboxID   string   `json:"sandbox_id"`
	Destination string   `json:"destination"`
	Items       []string `json:"items"`
}

// OutputPackResponse is the response for packing outputs.
type OutputPackResponse struct {
	SandboxID string `json:"sandbox_id"`
	Output    string `json:"output"`
	Size      int64  `json:"size"`
}

// OutputSandbox returns an HTTP handler for the single output endpoint.
// All deliverables (artifacts, build outputs, workspace patch) leave the
// sandbox through this channel; the data is read from inside the sandbox
// container, never from a host workspace directory.
func OutputSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sandboxID := mux.Vars(r)["id"]
		if sandboxID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sandbox ID is required"})
			return
		}

		// Verify sandbox exists
		meta, err := store.Get(sandboxID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}
		if err := requireCompat(meta); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		// Parse request
		var req OutputRequest
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
				return
			}
		}

		// Default action is "list"
		if req.Action == "" {
			req.Action = "list"
		}

		switch req.Action {
		case "list":
			handleOutputList(w, r, sandboxID)
		case "pull":
			handleOutputPull(w, r, sandboxID, req)
		case "pack":
			handleOutputPack(w, r, sandboxID, req)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown action: %s", req.Action)})
		}
	}
}

// existingOutputSources returns the output source dirs that exist and are
// non-empty inside the container.
func existingOutputSources(r *http.Request, sandboxID string) ([]string, error) {
	var existing []string
	for _, src := range outputSources {
		out, err := workspaceExec(r.Context(), sandboxID,
			"[ -d "+shellQuote(src)+" ] && ls -A "+shellQuote(src)+" 2>/dev/null | head -1 || true")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(out) != "" {
			existing = append(existing, src)
		}
	}
	return existing, nil
}

// handleOutputList lists available deliverables from a sandbox.
func handleOutputList(w http.ResponseWriter, r *http.Request, sandboxID string) {
	var items []OutputItem

	for _, src := range outputSources {
		// One find per source: path, size, mtime for regular files.
		out, err := workspaceExec(r.Context(), sandboxID,
			"[ -d "+shellQuote(src)+" ] && find "+shellQuote(src)+" -maxdepth 2 -type f -printf '%p\\t%s\\t%TY-%Tm-%TdT%TH:%TM:%TSZ\\n' 2>/dev/null || true")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list output: " + err.Error()})
			return
		}
		for _, line := range strings.Split(out, "\n") {
			parts := strings.SplitN(line, "\t", 3)
			if len(parts) != 3 {
				continue
			}
			size, _ := strconv.ParseInt(parts[1], 10, 64)
			items = append(items, OutputItem{
				Path:     parts[0],
				Type:     "file",
				Size:     size,
				Modified: parts[2],
			})
		}
	}

	// Workspace patch is a deliverable when the workspace has changes.
	if diff, err := workspaceDiff(r.Context(), sandboxID); err == nil && strings.TrimSpace(diff) != "" {
		items = append(items, OutputItem{
			Path: "/workspace.patch",
			Type: "patch",
			Size: int64(len(diff)),
		})
	}

	if items == nil {
		items = []OutputItem{}
	}
	writeJSON(w, http.StatusOK, OutputListResponse{
		SandboxID: sandboxID,
		Items:     items,
	})
}

// handleOutputPull delivers artifacts and the workspace patch to a host
// destination directory.
func handleOutputPull(w http.ResponseWriter, r *http.Request, sandboxID string, req OutputRequest) {
	if req.Dest == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "destination is required for pull"})
		return
	}

	if err := os.MkdirAll(req.Dest, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create destination: %v", err)})
		return
	}

	c, err := compatContainerHandle(sandboxID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	existing, err := existingOutputSources(r, sandboxID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var delivered []string
	for _, src := range existing {
		destPath := filepath.Join(req.Dest, strings.TrimPrefix(src, "/"))
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("prepare %s: %v", destPath, err)})
			return
		}
		if err := c.CopyFrom(src, destPath); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("copy %s: %v", src, err)})
			return
		}
		delivered = append(delivered, src)
	}

	// Deliver the workspace patch when there are changes.
	if diff, err := workspaceDiff(r.Context(), sandboxID); err == nil && strings.TrimSpace(diff) != "" {
		patchPath := filepath.Join(req.Dest, "workspace.patch")
		if err := os.WriteFile(patchPath, []byte(diff), 0644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("write patch: %v", err)})
			return
		}
		delivered = append(delivered, "/workspace.patch")
	}

	writeJSON(w, http.StatusOK, OutputPullResponse{
		SandboxID:   sandboxID,
		Destination: req.Dest,
		Items:       delivered,
	})
}

// handleOutputPack creates a compressed archive of the deliverables,
// built inside the container and copied to the host output path.
func handleOutputPack(w http.ResponseWriter, r *http.Request, sandboxID string, req OutputRequest) {
	if req.Output == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "output path is required for pack"})
		return
	}

	if dir := filepath.Dir(req.Output); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create output dir: %v", err)})
			return
		}
	}

	c, err := compatContainerHandle(sandboxID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	existing, err := existingOutputSources(r, sandboxID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if len(existing) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no output to pack"})
		return
	}

	// Copy sources to a host staging dir, then archive with the Go
	// stdlib. Building the archive inside the container is fragile:
	// /tmp is tmpfs (invisible to docker cp) and archiving a source
	// directory into itself makes tar fail.
	staging, err := os.MkdirTemp("", "pi-box-pack-*")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("stage archive: %v", err)})
		return
	}
	defer os.RemoveAll(staging)

	for _, src := range existing {
		destPath := filepath.Join(staging, strings.TrimPrefix(src, "/"))
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("stage %s: %v", src, err)})
			return
		}
		if err := c.CopyFrom(src, destPath); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("copy %s: %v", src, err)})
			return
		}
	}

	if err := tarGzDir(staging, req.Output); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create archive: %v", err)})
		return
	}

	info, err := os.Stat(req.Output)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("get archive size: %v", err)})
		return
	}

	writeJSON(w, http.StatusOK, OutputPackResponse{
		SandboxID: sandboxID,
		Output:    req.Output,
		Size:      info.Size(),
	})
}
