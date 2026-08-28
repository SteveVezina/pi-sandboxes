package compat

import (
	"context"
	"strings"
)

// ReconcileResult reports the outcome of container reconciliation
// (PROP-008 T15.2c).
type ReconcileResult struct {
	// MissingSandboxIDs are active sandbox IDs whose backend container
	// no longer exists (e.g. removed out-of-band while the daemon was
	// down). The caller is responsible for marking these DESTROYED.
	MissingSandboxIDs []string
	// RemovedContainers are container names removed because they had
	// no corresponding active sandbox record.
	RemovedContainers []string
}

// Reconcile compares live OCI containers against the daemon's active
// (non-terminal) sandbox IDs. It garbage-collects containers with no
// matching active sandbox and reports sandboxes whose container has
// vanished, so the caller can reconcile the sandbox store.
//
// If no OCI runtime is available, it returns a zero result rather than
// an error, since compat mode may not be in use on this host.
func Reconcile(ctx context.Context, activeSandboxIDs []string) (ReconcileResult, error) {
	eng, err := Engine()
	if err != nil {
		return ReconcileResult{}, nil
	}

	containers, err := eng.List(ctx)
	if err != nil {
		return ReconcileResult{}, err
	}

	activeNames := make(map[string]string, len(activeSandboxIDs))
	for _, id := range activeSandboxIDs {
		activeNames[ContainerName(id)] = id
	}

	seen := make(map[string]bool, len(containers))
	var result ReconcileResult
	for _, c := range containers {
		if !strings.HasPrefix(c.Name, "pi-sandbox-") {
			continue
		}
		seen[c.Name] = true
		if _, active := activeNames[c.Name]; active {
			continue
		}
		if err := eng.Remove(ctx, c.Name); err == nil {
			result.RemovedContainers = append(result.RemovedContainers, c.Name)
		}
	}

	for name, id := range activeNames {
		if !seen[name] {
			result.MissingSandboxIDs = append(result.MissingSandboxIDs, id)
		}
	}

	return result, nil
}
