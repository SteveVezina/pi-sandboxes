package session_test

import (
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestTTLChecker_ExpiresSession(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	checker := session.NewTTLChecker(store, 500*time.Millisecond)
	checker.Start()
	defer checker.Stop()

	// Let the checker tick once so it's settled
	time.Sleep(600 * time.Millisecond)

	// Create session with 1s TTL
	id, err := store.Create("test-box", "node-python", "fast")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Set TTL to 1 second and set LastUsedAt to now
	store.UpdateTTL(id, 1)
	store.UpdateLastUsed(id)
	store.UpdateState(id, session.StateWarm)

	// Wait for TTL check to run (1s TTL + buffer)
	time.Sleep(2 * time.Second)

	// Session should be transitioning to destroying
	meta, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.State != session.StateDestroying {
		t.Errorf("Expected state DESTROYING, got %s", meta.State)
	}
}

func TestTTLChecker_InfiniteTTL(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, _ := store.Create("test-box", "node-python", "fast")
	store.UpdateState(id, session.StateWarm)

	// TTL of 0 = infinite
	store.UpdateLastUsed(id)

	checker := session.NewTTLChecker(store, 500*time.Millisecond)
	checker.Start()
	defer checker.Stop()

	// Wait for multiple TTL checks
	time.Sleep(2 * time.Second)

	meta, _ := store.Get(id)
	if meta.State != session.StateWarm {
		t.Errorf("Expected state WARM (infinite TTL), got %s", meta.State)
	}
}
