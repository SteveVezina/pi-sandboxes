package system_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/system"
)

func TestPiHome(t *testing.T) {
	home := os.Getenv("HOME")
	expected := filepath.Join(home, ".pi")
	if system.PiHome() != expected {
		t.Errorf("Expected %s, got %s", expected, system.PiHome())
	}
}

func TestPiHome_NoHomeEnv(t *testing.T) {
	os.Unsetenv("HOME")
	if system.PiHome() != filepath.Join(".", ".pi") {
		t.Error("Expected ./\\.pi when HOME is unset")
	}
	os.Setenv("HOME", "/tmp")
}

func TestDirExists_True(t *testing.T) {
	if !system.DirExists("/tmp") {
		t.Error("Expected /tmp to exist")
	}
}

func TestDirExists_False(t *testing.T) {
	if system.DirExists("/nonexistent-dir-xyz") {
		t.Error("Expected /nonexistent-dir-xyz to not exist")
	}
}

func TestDirSize(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-sys-size-"+randomID())
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world12345"), 0644)

	size, err := system.DirSize(tmpDir)
	if err != nil {
		t.Fatalf("DirSize failed: %v", err)
	}
	if size != 5+10 {
		t.Errorf("Expected size 15, got %d", size)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes  int64
		expect string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1024 * 1024, "1.0 MiB"},
		{1024 * 1024 * 1024, "1.0 GiB"},
	}

	for _, tc := range tests {
		result := system.FormatSize(tc.bytes)
		if result != tc.expect {
			t.Errorf("FormatSize(%d) = %q, want %q", tc.bytes, result, tc.expect)
		}
	}
}

func TestGetDiskUsage(t *testing.T) {
	// Create a fake pi home at ~/.pi
	fakeHome := filepath.Join(os.TempDir(), "pi-sys-home-"+randomID())
	os.MkdirAll(fakeHome, 0755)
	defer os.RemoveAll(fakeHome)

	// Set HOME to fakeHome so PiHome() returns fakeHome/.pi
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", fakeHome)
	defer os.Setenv("HOME", origHome)

	// Now create the actual .pi directory
	realPiHome := filepath.Join(fakeHome, ".pi")
	os.MkdirAll(filepath.Join(realPiHome, "sandboxes", "test1"), 0755)
	os.MkdirAll(filepath.Join(realPiHome, "templates", "base"), 0755)
	os.WriteFile(filepath.Join(realPiHome, "sandboxes", "test1", "meta.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(realPiHome, "templates", "base", "template.yaml"), []byte("name: base"), 0644)

	info, err := system.GetDiskUsage()
	if err != nil {
		t.Fatalf("GetDiskUsage failed: %v", err)
	}

	if info.Sandboxes != 15 { // meta.json is 15 bytes
		t.Errorf("Expected sandboxes size 15, got %d", info.Sandboxes)
	}
	if info.Templates != 10 { // template.yaml is 10 bytes
		t.Errorf("Expected templates size 10, got %d", info.Templates)
	}
	if info.Total != 25 {
		t.Errorf("Expected total 23, got %d", info.Total)
	}
}

func TestDoctor(t *testing.T) {
	result := system.RunDoctor()
	if !result.Passed {
		t.Error("Expected doctor to pass")
	}

	// Should have at least one issue
	if len(result.Issues) == 0 {
		t.Error("Expected at least one issue")
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	if system.TimeAgo(now) != "just now" {
		t.Error("Expected 'just now' for current time")
	}

	minAgo := now.Add(-2 * time.Minute)
	if system.TimeAgo(minAgo) != "2m ago" {
		t.Errorf("Expected '2m ago', got %s", system.TimeAgo(minAgo))
	}

	hourAgo := now.Add(-3 * time.Hour)
	if system.TimeAgo(hourAgo) != "3h ago" {
		t.Errorf("Expected '3h ago', got %s", system.TimeAgo(hourAgo))
	}

	dayAgo := now.Add(-2 * 24 * time.Hour)
	if system.TimeAgo(dayAgo) != "2d ago" {
		t.Errorf("Expected '2d ago', got %s", system.TimeAgo(dayAgo))
	}
}

func TestStatus(t *testing.T) {
	// Create a fake pi home at ~/.pi
	fakeHome := filepath.Join(os.TempDir(), "pi-sys-home-status-"+randomID())
	os.MkdirAll(fakeHome, 0755)
	defer os.RemoveAll(fakeHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", fakeHome)
	defer os.Setenv("HOME", origHome)

	realPiHome := filepath.Join(fakeHome, ".pi")
	os.MkdirAll(filepath.Join(realPiHome, "sandboxes", "active1"), 0755)
	os.MkdirAll(filepath.Join(realPiHome, "sandboxes", "active2"), 0755)

	os.WriteFile(filepath.Join(realPiHome, "sandboxes", "active1", "meta.json"),
		[]byte(`{"name":"active1","state":"WARM"}`), 0644)
	os.WriteFile(filepath.Join(realPiHome, "sandboxes", "active2", "meta.json"),
		[]byte(`{"name":"active2","state":"EXECUTING"}`), 0644)

	info, err := system.GetStatus("")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if info.TotalSandboxes != 2 {
		t.Errorf("Expected 2 sandboxes, got %d", info.TotalSandboxes)
	}
	if info.ActiveSandboxes != 2 {
		t.Errorf("Expected 2 active sandboxes, got %d", info.ActiveSandboxes)
	}
}

func TestStatus_NoPiHome(t *testing.T) {
	os.Unsetenv("HOME")
	info, err := system.GetStatus("")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if info.PiHomeExists {
		t.Error("Expected PiHomeExists to be false")
	}
	os.Setenv("HOME", "/tmp")
}

func TestContainsString(t *testing.T) {
	if !system.ContainsString([]byte(`{"state":"destroyed"}`), `"state":"destroyed"`) {
		t.Error("Expected true for matching substring")
	}
	if system.ContainsString([]byte(`{"state":"WARM"}`), `"state":"destroyed"`) {
		t.Error("Expected false for non-matching substring")
	}
}

func TestPrune(t *testing.T) {
	fakeHome := filepath.Join(os.TempDir(), "pi-sys-home-prune-"+randomID())
	os.MkdirAll(fakeHome, 0755)
	defer os.RemoveAll(fakeHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", fakeHome)
	defer os.Setenv("HOME", origHome)

	realPiHome := filepath.Join(fakeHome, ".pi")
	os.MkdirAll(filepath.Join(realPiHome, "sandboxes", "destroyed1"), 0755)
	os.WriteFile(filepath.Join(realPiHome, "sandboxes", "destroyed1", "meta.json"),
		[]byte(`{"name":"destroyed1","state":"destroyed"}`), 0644)

	result, err := system.RunPrune(false)
	if err != nil {
		t.Fatalf("RunPrune failed: %v", err)
	}

	if result.RemovedSandboxes != 1 {
		t.Errorf("Expected 1 removed sandbox, got %d", result.RemovedSandboxes)
	}

	// Verify it was removed
	if system.DirExists(filepath.Join(realPiHome, "sandboxes", "destroyed1")) {
		t.Error("Expected destroyed sandbox to be removed")
	}
}

func TestDiskUsage_Empty(t *testing.T) {
	fakeHome := filepath.Join(os.TempDir(), "pi-sys-home-empty-"+randomID())
	os.MkdirAll(fakeHome, 0755)
	defer os.RemoveAll(fakeHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", fakeHome)
	defer os.Setenv("HOME", origHome)

	// Don't create .pi — should return 0

	info, err := system.GetDiskUsage()
	if err != nil {
		t.Fatalf("GetDiskUsage failed: %v", err)
	}

	if info.Total != 0 {
		t.Errorf("Expected 0 total for empty pi home, got %d", info.Total)
	}
}

func randomID() string {
	b := []byte("abcdefghijklmnopqrstuvwxyz012345")
	n := len(b)
	result := make([]byte, 8)
	for i := range result {
		result[i] = b[i%n]
	}
	return string(result)
}

// Export the containsString function for testing
func init() {
	// Make sure ContainsString is exported
	_ = fmt.Sprintf // suppress unused import
}
