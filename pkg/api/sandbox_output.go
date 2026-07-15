package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

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
	SandboxID string   `json:"sandbox_id"`
	Destination string `json:"destination"`
	Items     []string `json:"items"`
}

// OutputPackResponse is the response for packing outputs.
type OutputPackResponse struct {
	SandboxID string `json:"sandbox_id"`
	Output    string `json:"output"`
	Size      int64  `json:"size"`
}

// OutputSandbox returns an HTTP handler for the single output endpoint.
func OutputSandbox(store *sandbox.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sandboxID := r.PathValue("id")
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
			handleOutputList(w, sandboxID, meta)
		case "pull":
			handleOutputPull(w, r, sandboxID, meta, req)
		case "pack":
			handleOutputPack(w, sandboxID, meta, req)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown action: %s", req.Action)})
		}
	}
}

// handleOutputList lists available deliverables from a sandbox.
func handleOutputList(w http.ResponseWriter, sandboxID string, meta *sandbox.Meta) {
	// Known output sources from SPEC.md
	outputSources := map[string]string{
		"/artifacts":           "primary artifacts",
		"/workspace/dist":      "build outputs",
		"/workspace/build":     "build outputs",
		"/workspace/coverage":  "test coverage reports",
		"/workspace/test-results": "test result files",
		"/workspace/target/release": "Rust release binaries",
	}

	var items []OutputItem

	// Scan known output sources
	for srcPath := range outputSources {
		fullPath := filepath.Join("/sandbox", sandboxID, srcPath)
		if info, err := os.Stat(fullPath); err == nil {
			if info.IsDir() {
				// List files in directory
				entries, err := os.ReadDir(fullPath)
				if err == nil {
					for _, entry := range entries {
						if !entry.IsDir() {
							itemInfo, err := entry.Info()
							if err == nil {
								items = append(items, OutputItem{
									Path:     filepath.Join(srcPath, entry.Name()),
									Type:     "file",
									Size:     itemInfo.Size(),
									Modified: itemInfo.ModTime().Format(time.RFC3339),
								})
							}
						}
					}
				}
			} else {
				items = append(items, OutputItem{
					Path:     srcPath,
					Type:     "file",
					Size:     info.Size(),
					Modified: info.ModTime().Format(time.RFC3339),
				})
			}
		}
	}

	// Check for workspace patch
	if hasWorkspaceChanges(sandboxID) {
		items = append(items, OutputItem{
			Path: "/workspace.patch",
			Type: "patch",
			Size: 0, // Will be calculated on pull
			Modified: time.Now().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, OutputListResponse{
		SandboxID: sandboxID,
		Items:     items,
	})
}

// handleOutputPull delivers artifacts or patches to a host destination.
func handleOutputPull(w http.ResponseWriter, r *http.Request, sandboxID string, meta *sandbox.Meta, req OutputRequest) {
	if req.Dest == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "destination is required for pull"})
		return
	}

	// Create destination directory
	if err := os.MkdirAll(req.Dest, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create destination: %v", err)})
		return
	}

	// Known output sources
	outputSources := []string{
		"/artifacts",
		"/workspace/dist",
		"/workspace/build",
		"/workspace/coverage",
		"/workspace/test-results",
		"/workspace/target/release",
	}

	var delivered []string

	// Copy known output sources
	for _, srcPath := range outputSources {
		srcDir := filepath.Join("/sandbox", sandboxID, srcPath)
		destPath := filepath.Join(req.Dest, srcPath[1:]) // Remove leading /

		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			// Copy directory contents
			if err := copyDir(srcDir, destPath); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("copy %s: %v", srcPath, err)})
				return
			}
			delivered = append(delivered, srcPath)
		}
	}

	// Check for workspace patch
	if hasWorkspaceChanges(sandboxID) {
		patchPath := filepath.Join(req.Dest, "workspace.patch")
		if err := generatePatch(sandboxID, patchPath); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("generate patch: %v", err)})
			return
		}
		delivered = append(delivered, "/workspace.patch")
	}

	// Emit pi.artifact.delivered event (in production, this would be emitted via the daemon)
	_ = meta // In production, emit event here

	writeJSON(w, http.StatusOK, OutputPullResponse{
		SandboxID:     sandboxID,
		Destination:   req.Dest,
		Items:         delivered,
	})
}

// handleOutputPack creates a compressed archive of selected output sources.
func handleOutputPack(w http.ResponseWriter, sandboxID string, meta *sandbox.Meta, req OutputRequest) {
	if req.Output == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "output path is required for pack"})
		return
	}

	// Create parent directory if needed
	if dir := filepath.Dir(req.Output); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create output dir: %v", err)})
			return
		}
	}

	// Known output sources
	outputSources := []string{
		"/artifacts",
		"/workspace/dist",
		"/workspace/build",
		"/workspace/coverage",
		"/workspace/test-results",
		"/workspace/target/release",
	}

	var collected []string

	// Collect known output sources
	for _, srcPath := range outputSources {
		srcDir := filepath.Join("/sandbox", sandboxID, srcPath)
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			collected = append(collected, srcPath)
		}
	}

	// Create tar.zst archive (simplified - in production, use proper tar+zstd)
	if err := createArchive(req.Output, collected); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("create archive: %v", err)})
		return
	}

	// Get archive size
	info, err := os.Stat(req.Output)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("get archive size: %v", err)})
		return
	}

	// Emit pi.artifact.delivered event
	_ = meta // In production, emit event here

	writeJSON(w, http.StatusOK, OutputPackResponse{
		SandboxID: sandboxID,
		Output:    req.Output,
		Size:      info.Size(),
	})
}

// hasWorkspaceChanges checks if the workspace has changes.
func hasWorkspaceChanges(sandboxID string) bool {
	// In production, check the workspace diff
	return false
}

// generatePatch generates a workspace patch.
func generatePatch(sandboxID, destPath string) error {
	// In production, generate actual patch
	return os.WriteFile(destPath, []byte("# Patch placeholder"), 0644)
}

// copyDir copies a directory recursively.
func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}

// copyFile copies a single file.
func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// createArchive creates a tar.zst archive (simplified).
func createArchive(outputPath string, sources []string) error {
	// In production, use proper tar+zstd implementation
	return os.WriteFile(outputPath, []byte("archive placeholder"), 0644)
}

// sanitizePath validates and sanitizes a file path to prevent directory traversal.
func sanitizePath(base, requested string) (string, error) {
	cleaned := filepath.Clean(filepath.Join(base, requested))
	if !strings.HasPrefix(cleaned, base) {
		return "", fmt.Errorf("path traversal detected")
	}
	return cleaned, nil
}
