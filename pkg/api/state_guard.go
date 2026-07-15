package api

import (
	"net/http"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func requireSandboxState(w http.ResponseWriter, meta *sandbox.Meta, allowed ...sandbox.State) bool {
	for _, state := range allowed {
		if meta.State == state {
			return true
		}
	}
	writeJSON(w, http.StatusConflict, map[string]string{
		"error":      "sandbox is not in a ready state for this operation",
		"state":      string(meta.State),
		"required":   requiredStatesLabel(allowed),
		"sandbox_id": meta.ID,
	})
	return false
}

func requiredStatesLabel(states []sandbox.State) string {
	if len(states) == 0 {
		return ""
	}
	label := string(states[0])
	for _, state := range states[1:] {
		label += "," + string(state)
	}
	return label
}
