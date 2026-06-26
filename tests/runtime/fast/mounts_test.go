package fast_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/fast"
)

func TestDefaultMountConfig(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-mount-test")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	cfg := fast.DefaultMountConfig(tmpDir, "test-sandbox")

	if cfg.Workspace != filepath.Join(tmpDir, "sandboxes", "test-sandbox", "workspace") {
		t.Error("Expected correct workspace path")
	}
	if cfg.Artifacts != filepath.Join(tmpDir, "sandboxes", "test-sandbox", "artifacts") {
		t.Error("Expected correct artifacts path")
	}
	if len(cfg.Caches) != 7 {
		t.Errorf("Expected 7 cache dirs, got %d", len(cfg.Caches))
	}
}

func TestMountConfig_EnsureDirectories(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-mount-test2")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	cfg := fast.DefaultMountConfig(tmpDir, "test-sandbox")

	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories failed: %v", err)
	}

	// Verify directories exist
	entries, _ := os.ReadDir(tmpDir)
	found := make(map[string]bool)
	for _, e := range entries {
		found[e.Name()] = true
	}

	expectedDirs := []string{"rootfs", "sandboxes", "caches"}
	for _, d := range expectedDirs {
		if !found[d] {
			t.Errorf("Expected directory %s to exist", d)
		}
	}
}

func TestMountConfig_ValidateNoHostMounts(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-mount-test3")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	cfg := fast.DefaultMountConfig(tmpDir, "test-sandbox")

	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected no error with no host mounts, got: %v", err)
	}
}

func TestMountConfig_RejectsHostMounts(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-mount-test4")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	cfg := fast.DefaultMountConfig(tmpDir, "test-sandbox")
	cfg.HostMounts = []fast.HostMount{
		{HostPath: "/host", ContainerPath: "/workspace"},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Expected error with host mounts, got nil")
	}
}

func TestMountConfig_Directories(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-mount-test5")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)

	cfg := fast.DefaultMountConfig(tmpDir, "test-sandbox")

	if cfg.WorkspaceDir() != cfg.Workspace {
		t.Error("WorkspaceDir() should return Workspace")
	}
	if cfg.ArtifactsDir() != cfg.Artifacts {
		t.Error("ArtifactsDir() should return Artifacts")
	}
}
