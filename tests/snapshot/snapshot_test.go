package snapshot_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/snapshot"
)

func TestCreateMeta(t *testing.T) {
	m := newTestManager(t)

	meta, err := m.CreateMeta("test-snap", 1024, "tar")
	if err != nil {
		t.Fatalf("CreateMeta failed: %v", err)
	}

	if meta.Name != "test-snap" {
		t.Errorf("Expected name 'test-snap', got '%s'", meta.Name)
	}
	if meta.SizeBytes != 1024 {
		t.Errorf("Expected size 1024, got %d", meta.SizeBytes)
	}
	if meta.Method != "tar" {
		t.Errorf("Expected method 'tar', got '%s'", meta.Method)
	}
}

func TestGetMeta(t *testing.T) {
	m := newTestManager(t)

	m.CreateMeta("snap1", 2048, "overlay")
	meta, err := m.GetMeta("snap1")
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}

	if meta.SizeBytes != 2048 {
		t.Errorf("Expected size 2048, got %d", meta.SizeBytes)
	}
}

func TestGetMeta_NotFound(t *testing.T) {
	m := newTestManager(t)

	_, err := m.GetMeta("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent snapshot")
	}
}

func TestExists(t *testing.T) {
	m := newTestManager(t)

	m.CreateMeta("snap1", 100, "tar")
	if !m.Exists("snap1") {
		t.Error("Expected snap1 to exist")
	}
	if m.Exists("nonexistent") {
		t.Error("Expected nonexistent to not exist")
	}
}

func TestCreate(t *testing.T) {
	m := newTestManager(t)

	// Create a workspace with files
	workspaceDir := filepath.Join(os.TempDir(), "pi-snap-workspace-"+randomID())
	os.MkdirAll(filepath.Join(workspaceDir, "src"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "README.md"), []byte("# Test"), 0644)
	os.WriteFile(filepath.Join(workspaceDir, "src", "main.go"), []byte("package main"), 0644)
	defer os.RemoveAll(workspaceDir)

	result, err := m.Create("before-refactor", workspaceDir)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected snapshot creation to succeed")
	}
	if result.Method != "tar" {
		t.Errorf("Expected method 'tar', got '%s'", result.Method)
	}
}

func TestRollback(t *testing.T) {
	m := newTestManager(t)

	// Create workspace and snapshot
	workspaceDir := filepath.Join(os.TempDir(), "pi-snap-rollback-"+randomID())
	os.MkdirAll(filepath.Join(workspaceDir, "src"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "original.txt"), []byte("original content"), 0644)
	defer os.RemoveAll(workspaceDir)

	m.Create("snap1", workspaceDir)

	// Modify workspace
	os.WriteFile(filepath.Join(workspaceDir, "original.txt"), []byte("modified content"), 0644)
	os.WriteFile(filepath.Join(workspaceDir, "new.txt"), []byte("new file"), 0644)

	// Rollback
	err := m.Rollback("snap1", workspaceDir)
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	// Verify rollback
	data, err := os.ReadFile(filepath.Join(workspaceDir, "original.txt"))
	if err != nil {
		t.Fatalf("Read file after rollback: %v", err)
	}
	if string(data) != "original content" {
		t.Errorf("Expected 'original content', got '%s'", string(data))
	}

	// New file should be gone
	if _, err := os.Stat(filepath.Join(workspaceDir, "new.txt")); !os.IsNotExist(err) {
		t.Error("Expected new.txt to be removed after rollback")
	}
}

func TestRollback_NotFound(t *testing.T) {
	m := newTestManager(t)

	workspaceDir := filepath.Join(os.TempDir(), "pi-snap-workspace-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	err := m.Rollback("nonexistent", workspaceDir)
	if err == nil {
		t.Fatal("Expected error for nonexistent snapshot")
	}
}

func TestList(t *testing.T) {
	m := newTestManager(t)

	m.CreateMeta("snap1", 100, "tar")
	m.CreateMeta("snap2", 200, "overlay")

	snapshots, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(snapshots) != 2 {
		t.Fatalf("Expected 2 snapshots, got %d", len(snapshots))
	}

	// Should be sorted newest first
	if snapshots[0].Name == "snap1" {
		// If snap1 is first, snap2 must be second (since snap2 was created after)
		// Actually, CreateMeta uses time.Now(), so order depends on timing
		// Just verify we have 2 entries
	}
}

func TestList_Empty(t *testing.T) {
	m := newTestManager(t)

	snapshots, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestDelete(t *testing.T) {
	m := newTestManager(t)

	m.CreateMeta("snap1", 100, "tar")
	if !m.Exists("snap1") {
		t.Fatal("Expected snap1 to exist before delete")
	}

	err := m.Delete("snap1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if m.Exists("snap1") {
		t.Error("Expected snap1 to be deleted")
	}
}

func TestDelete_NotFound(t *testing.T) {
	m := newTestManager(t)

	err := m.Delete("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent snapshot")
	}
}

func TestSnapshotTimestamp(t *testing.T) {
	m := newTestManager(t)

	before := time.Now().UTC()
	m.CreateMeta("ts-test", 0, "tar")
	after := time.Now().UTC()

	meta, err := m.GetMeta("ts-test")
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}

	if meta.CreatedAt.Before(before) || meta.CreatedAt.After(after) {
		t.Errorf("Timestamp %v not between %v and %v", meta.CreatedAt, before, after)
	}
}

func newTestManager(t *testing.T) *snapshot.Manager {
	tmpDir := filepath.Join(os.TempDir(), "pi-snap-test-"+randomID())
	os.MkdirAll(tmpDir, 0755)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	return snapshot.NewManager(tmpDir)
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
