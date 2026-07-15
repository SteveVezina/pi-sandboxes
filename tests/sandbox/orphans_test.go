package session_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestOrphanCleanup_CleansOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	// Create and mark as EXECUTING (no real process = orphan)
	id, _ := store.Create("orphan-box", "node-python", "fast")
	store.UpdateState(id, session.StateExecuting)

	// Run orphan cleanup
	session.OrphanCleanup(store, tmpDir)

	// Should be marked DESTROYED
	meta, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.State != session.StateDestroyed {
		t.Errorf("Expected state DESTROYED, got %s", meta.State)
	}
}

func TestOrphanCleanup_SkipsWarmSessions(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, _ := store.Create("warm-box", "node-python", "fast")
	store.UpdateState(id, session.StateWarm)

	session.OrphanCleanup(store, tmpDir)

	meta, _ := store.Get(id)
	if meta.State != session.StateWarm {
		t.Errorf("Expected state WARM (not cleaned), got %s", meta.State)
	}
}

func TestOrphanCleanup_SkipsDestroyed(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, _ := store.Create("destroyed-box", "node-python", "fast")
	store.UpdateState(id, session.StateDestroyed)

	session.OrphanCleanup(store, tmpDir)

	meta, _ := store.Get(id)
	if meta.State != session.StateDestroyed {
		t.Errorf("Expected state DESTROYED (unchanged), got %s", meta.State)
	}
}
