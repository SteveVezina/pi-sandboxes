package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pi-sandbox/pi/pkg/logs"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// LogsSandbox returns an HTTP handler for log operations.
func LogsSandbox(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		// Validate sandbox exists
		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		lm := logs.NewManager(id)

		// Get action from query or body
		action := r.URL.Query().Get("action")
		if action == "" {
			action = "list"
		}

		switch action {
		case "list", "":
			// List all log entries
			entries, err := lm.List()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list logs: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":      id,
				"action":  "list",
				"count":   len(entries),
				"entries": entries,
			})

		case "history":
			// Return history summary
			history, err := lm.History()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get history: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":      id,
				"action":  "history",
				"count":   len(history),
				"entries": history,
			})

		case "get":
			// Get a specific entry
			seqStr := r.URL.Query().Get("sequence")
			if seqStr == "" {
				var req struct {
					Sequence int `json:"sequence"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
					seqStr = ""
					_ = req
				}
			}
			// For now, just list all
			entries, err := lm.List()
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get log: " + err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":      id,
				"action":  "get",
				"entries": entries,
			})

		case "stdout":
			// Get stdout for a specific entry
			seqStr := r.URL.Query().Get("sequence")
			if seqStr == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sequence parameter required"})
				return
			}
			// Parse sequence (simplified — in production use strconv)
			var seq int
			_ = json.Unmarshal([]byte(seqStr), &seq)
			if seq == 0 {
				seq, _ = parseInt(seqStr)
			}
			stdout, err := lm.GetStdout(seq)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "get stdout: " + err.Error()})
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(stdout))

		case "stderr":
			// Get stderr for a specific entry
			seqStr := r.URL.Query().Get("sequence")
			if seqStr == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sequence parameter required"})
				return
			}
			var seq int
			_ = json.Unmarshal([]byte(seqStr), &seq)
			if seq == 0 {
				seq, _ = parseInt(seqStr)
			}
			stderr, err := lm.GetStderr(seq)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "get stderr: " + err.Error()})
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(stderr))

		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action: " + action})
		}
	}
}

// LogsList returns an HTTP handler that lists all log entries.
func LogsList(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		lm := logs.NewManager(id)
		entries, err := lm.List()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list logs: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"count":   len(entries),
			"entries": entries,
		})
	}
}

// LogsHistory returns an HTTP handler that returns command history.
func LogsHistory(store *session.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := store.Get(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "sandbox not found"})
			return
		}

		lm := logs.NewManager(id)
		history, err := lm.History()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "get history: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"count":   len(history),
			"entries": history,
		})
	}
}

// parseInt is a simplified int parser for query params.
func parseInt(s string) (int, error) {
	var result int
	var neg bool
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	if neg {
		result = -result
	}
	return result, nil
}
