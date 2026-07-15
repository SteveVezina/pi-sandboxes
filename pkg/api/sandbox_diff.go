package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// gitPreamble fails fast with a clear message when the sandbox image has
// no git, and marks /workspace safe for the container user.
const gitPreamble = "command -v git >/dev/null 2>&1 || { echo 'git is not available in this sandbox image' >&2; exit 127; }; " +
	"git config --global --add safe.directory /workspace >/dev/null 2>&1; "

// workspaceDiff produces the git diff of /workspace inside the container.
func workspaceDiff(ctx context.Context, id string) (string, error) {
	script := gitPreamble +
		"cd /workspace && if git rev-parse --git-dir >/dev/null 2>&1; then " +
		"git add -N . >/dev/null 2>&1; git diff HEAD 2>/dev/null || git diff; " +
		"else echo 'workspace is not a git repository' >&2; exit 1; fi"
	return workspaceExec(ctx, id, script)
}

// DiffSandbox returns an HTTP handler that shows the workspace diff
// computed inside the sandbox container.
func DiffSandbox(store *sandbox.Store) http.HandlerFunc {
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

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		start := time.Now()
		diff, err := workspaceDiff(ctx, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "diff failed: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":          id,
			"name":        meta.Name,
			"diff":        diff,
			"timed_out":   ctx.Err() == context.DeadlineExceeded,
			"duration_ms": time.Since(start).Milliseconds(),
		})
	}
}
