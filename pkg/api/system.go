package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pi-sandbox/pi/pkg/runtime/detect"
	"github.com/pi-sandbox/pi/pkg/session"
	"github.com/pi-sandbox/pi/pkg/system"
)

// SystemStatus returns GUI-readable local daemon status.
func SystemStatus(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, err := store.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		active := 0
		for _, id := range ids {
			meta, err := store.Get(id)
			if err == nil && (meta.State == session.StateWarm || meta.State == session.StateExecuting) {
				active++
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"daemon":           "connected",
			"active_sandboxes": active,
			"total_sandboxes":  len(ids),
			"pi_home":          system.PiHome(),
			"config_path":      filepath.Join(system.PiHome(), "config.yaml"),
			"support_redacted": true,
		})
	}
}

// SystemDoctor returns doctor-equivalent diagnostics for GUI display.
func SystemDoctor() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := system.RunDoctor()
		writeJSON(w, http.StatusOK, result)
	}
}

// SystemRuntimes returns available runtime backend information.
func SystemRuntimes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		available := detect.AvailableRuntimes("")
		runtimes := detect.AllRuntimes("")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"available":  available,
			"best":       detect.BestMode(""),
			"backends":   runtimes,
		})
	}
}

// SupportBundle returns a redacted support bundle payload for the GUI.
func SupportBundle(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids, _ := store.List()
		doctor := redactedDoctor(system.RunDoctor())
		available := detect.AvailableRuntimes("")

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"version": map[string]string{
				"component": "pi-sandboxd",
			},
			"diagnostics": doctor,
			"runtimes": map[string]interface{}{
				"available": available,
				"best":      detect.BestMode(""),
			},
			"sessions": map[string]interface{}{
				"count": len(ids),
				"ids":   ids,
			},
			"config": map[string]string{
				"path":    redactPath(filepath.Join(system.PiHome(), "config.yaml")),
				"pi_home": redactPath(system.PiHome()),
			},
			"redacted": true,
		})
	}
}

func redactPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		path = strings.Replace(path, home, "~", 1)
	}
	return path
}

func redactedDoctor(result *system.DoctorResult) *system.DoctorResult {
	if result == nil {
		return result
	}
	redacted := &system.DoctorResult{Passed: result.Passed}
	for _, issue := range result.Issues {
		issue.Message = redactPath(issue.Message)
		issue.Recommendation = redactPath(issue.Recommendation)
		redacted.Issues = append(redacted.Issues, issue)
	}
	return redacted
}
