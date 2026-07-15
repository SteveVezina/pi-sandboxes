package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestStore_Create(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, err := store.Create("test-box", "node-python", "fast")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id == "" {
		t.Fatal("Create returned empty ID")
	}

	// Verify meta.json exists
	metaPath := filepath.Join(tmpDir, id, "meta.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatal("meta.json not created")
	}
}

func TestStore_Create_DirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, err := store.Create("test-box", "node-python", "fast")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dir := filepath.Join(tmpDir, id)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	// Expect 0755
	expected := os.FileMode(0755)
	if info.Mode().Perm() != expected {
		t.Errorf("Expected permissions %o, got %o", expected, info.Mode().Perm())
	}
}

func TestStore_Get(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, err := store.Create("test-box", "node-python", "fast")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	meta, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.Name != "test-box" {
		t.Errorf("Expected name 'test-box', got '%s'", meta.Name)
	}
	if meta.Template != "node-python" {
		t.Errorf("Expected template 'node-python', got '%s'", meta.Template)
	}
	if meta.Mode != "fast" {
		t.Errorf("Expected mode 'fast', got '%s'", meta.Mode)
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	_, err := store.Get("nonexistent-id")
	if err == nil {
		t.Fatal("Expected error for nonexistent ID, got nil")
	}
}

func TestStore_Update(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, err := store.Create("test-box", "node-python", "fast")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.UpdateState(id, session.StateExecuting)
	if err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	meta, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if meta.State != session.StateExecuting {
		t.Errorf("Expected state 'EXECUTING', got '%s'", meta.State)
	}
}

func TestStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id1, _ := store.Create("box-1", "base", "fast")
	id2, _ := store.Create("box-2", "node", "fast")

	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(ids))
	}

	// Verify both IDs are in the list
	found := make(map[string]bool)
	for _, id := range ids {
		found[id] = true
	}
	if !found[id1] {
		t.Errorf("Expected ID %s in list", id1)
	}
	if !found[id2] {
		t.Errorf("Expected ID %s in list", id2)
	}
}

func TestStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, err := store.Create("test-box", "node-python", "fast")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.Delete(id)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify directory removed
	dir := filepath.Join(tmpDir, id)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("Session directory not removed after Delete")
	}

	// Verify Get fails
	_, err = store.Get(id)
	if err == nil {
		t.Fatal("Expected error for deleted session, got nil")
	}
}

func TestStore_Delete_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	store := session.NewStore(tmpDir)

	id, _ := store.Create("test-box", "node-python", "fast")

	// First delete should succeed
	err := store.Delete(id)
	if err != nil {
		t.Fatalf("First Delete failed: %v", err)
	}

	// Second delete should not error (idempotent)
	err = store.Delete(id)
	if err != nil {
		t.Fatalf("Second Delete should be idempotent, got: %v", err)
	}
}
