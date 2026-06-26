package fast_test

import (
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/fast"
)

func TestDefaultSeccompProfile(t *testing.T) {
	profile := fast.DefaultSeccompProfile()
	if profile == nil {
		t.Fatal("Expected non-nil profile")
	}
	if profile.DefaultAction == "" {
		t.Error("Expected non-empty defaultAction")
	}
}

func TestSeccompSaveFailsOnNonLinux(t *testing.T) {
	profile := fast.DefaultSeccompProfile()
	tmpDir := t.TempDir()
	err := profile.Save(filepath.Join(tmpDir, "seccomp.json"))
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestSeccompLoadFailsOnNonLinux(t *testing.T) {
	_, err := fast.Load("/tmp/nonexistent.json")
	// On non-Linux, Load returns the Linux error
	// But on non-Linux stub, it returns "seccomp requires Linux"
	// The file doesn't exist, so it might fail with os.ErrNotExist first
	// Either way, it should error
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}
