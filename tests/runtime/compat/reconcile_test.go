package compat_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/compat"
)

// TestReconcile_RemovesOrphanAndReportsMissing verifies the PROP-008
// T15.2c lifecycle recovery contract: containers with no matching active
// sandbox are garbage-collected, and active sandboxes whose container has
// vanished are reported so the daemon can reconcile its store.
func TestReconcile_RemovesOrphanAndReportsMissing(t *testing.T) {
	dir := t.TempDir()
	rmLog := filepath.Join(dir, "rm.log")

	fake := filepath.Join(dir, "docker")
	script := `#!/bin/sh
case "$1" in
  ps)
    echo "cid-a|pi-sandbox-aaaaaaaa|Up 2 minutes|debian:bookworm-slim"
    echo "cid-b|pi-sandbox-orphanid|Exited (0) 1 hour ago|debian:bookworm-slim"
    ;;
  rm)
    echo "$3" >> "` + rmLog + `"
    ;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	t.Setenv("PATH", dir)
	compat.ResetDetection()
	t.Cleanup(compat.ResetDetection)

	// aaaaaaaa-... has a live container (pi-sandbox-aaaaaaaa) and should
	// be left alone. bbbbbbbb-... has no container and should be
	// reported missing. The orphanid container has no matching active
	// sandbox and should be removed.
	activeIDs := []string{"aaaaaaaa-real-sandbox", "bbbbbbbb-no-container"}

	result, err := compat.Reconcile(context.Background(), activeIDs)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	if len(result.RemovedContainers) != 1 || result.RemovedContainers[0] != "pi-sandbox-orphanid" {
		t.Errorf("RemovedContainers = %v, want [pi-sandbox-orphanid]", result.RemovedContainers)
	}
	if len(result.MissingSandboxIDs) != 1 || result.MissingSandboxIDs[0] != "bbbbbbbb-no-container" {
		t.Errorf("MissingSandboxIDs = %v, want [bbbbbbbb-no-container]", result.MissingSandboxIDs)
	}

	logBytes, err := os.ReadFile(rmLog)
	if err != nil {
		t.Fatalf("read rm log: %v", err)
	}
	log := strings.TrimSpace(string(logBytes))
	if log != "pi-sandbox-orphanid" {
		t.Errorf("docker rm called with %q, want only pi-sandbox-orphanid (aaaaaaaa container must survive)", log)
	}
}

// TestReconcile_NoRuntimeAvailable verifies Reconcile degrades gracefully
// (zero result, no error) when no OCI runtime is on PATH, since compat
// mode may not be in use on this host.
func TestReconcile_NoRuntimeAvailable(t *testing.T) {
	dir := t.TempDir() // empty PATH, no docker/podman/runc binaries
	t.Setenv("PATH", dir)
	compat.ResetDetection()
	t.Cleanup(compat.ResetDetection)

	result, err := compat.Reconcile(context.Background(), []string{"any-id"})
	if err != nil {
		t.Fatalf("Reconcile returned error, want nil: %v", err)
	}
	if len(result.RemovedContainers) != 0 || len(result.MissingSandboxIDs) != 0 {
		t.Errorf("expected zero result with no runtime, got %+v", result)
	}
}
