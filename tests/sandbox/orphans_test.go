package sandbox_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestOrphanCleanup_CleansOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	store := sandbox.NewStore(tmpDir)

	// Create and mark as EXECUTING (no real process = orphan)
	id, _ := store.Create("orphan-box", "node-python", "fast")
	store.UpdateState(id, sandbox.StateExecuting)

	// Run orphan cleanup
	sandbox.OrphanCleanup(store, tmpDir)

	// Should be marked DESTROYED
	meta, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.State != sandbox.StateDestroyed {
		t.Errorf("Expected state DESTROYED, got %s", meta.State)
	}
}

func TestOrphanCleanup_SkipsWarmSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store := sandbox.NewStore(tmpDir)

	id, _ := store.Create("warm-box", "node-python", "fast")
	store.UpdateState(id, sandbox.StateWarm)

	sandbox.OrphanCleanup(store, tmpDir)

	meta, _ := store.Get(id)
	if meta.State != sandbox.StateWarm {
		t.Errorf("Expected state WARM (not cleaned), got %s", meta.State)
	}
}

func TestOrphanCleanup_SkipsDestroyed(t *testing.T) {
	tmpDir := t.TempDir()
	store := sandbox.NewStore(tmpDir)

	id, _ := store.Create("destroyed-box", "node-python", "fast")
	store.UpdateState(id, sandbox.StateDestroyed)

	sandbox.OrphanCleanup(store, tmpDir)

	meta, _ := store.Get(id)
	if meta.State != sandbox.StateDestroyed {
		t.Errorf("Expected state DESTROYED (unchanged), got %s", meta.State)
	}
}
