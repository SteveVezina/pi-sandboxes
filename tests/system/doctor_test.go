package system_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/system"
)

// TestDoctor_ConfigCreatedWhenMissing verifies RunDoctor creates config.yaml
// in a temp HOME when it doesn't exist (AC-16 / F10).
func TestDoctor_ConfigCreatedWhenMissing(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".pi"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("HOME", home)

	result := system.RunDoctor()

	// Config should now exist.
	configPath := filepath.Join(home, ".pi", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config.yaml was not created: %v", err)
	}

	// At least one issue should mention the config creation.
	found := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "Config file") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no issue mentioning config file; issues: %v", result.Issues)
	}
}

// TestDoctor_ConfigOKWhenPresent verifies RunDoctor reports OK for an existing
// valid config.yaml (AC-16 / F10).
func TestDoctor_ConfigOKWhenPresent(t *testing.T) {
	home := t.TempDir()
	piDir := filepath.Join(home, ".pi")
	if err := os.MkdirAll(piDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(piDir, "config.yaml"),
		[]byte("mode: fast\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("HOME", home)

	result := system.RunDoctor()

	found := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "Config file OK") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Config file OK' issue; got: %v", result.Issues)
	}
}

// TestDoctor_SystemCommandsValidated verifies RunDoctor checks for required
// system commands (AC-16 / F10 CORE.md watch-out).
func TestDoctor_SystemCommandsValidated(t *testing.T) {
	// 'find' is universally available; we just need at least one command check.
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".pi"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("HOME", home)

	result := system.RunDoctor()

	foundCmdCheck := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "Command available:") ||
			strings.Contains(iss.Message, "Command not found:") {
			foundCmdCheck = true
			break
		}
	}
	if !foundCmdCheck {
		t.Errorf("RunDoctor did not validate system commands; issues: %v", result.Issues)
	}
}

// TestDoctor_DaemonBinaryCheck verifies RunDoctor checks pi-sandboxd (AC-16).
func TestDoctor_DaemonBinaryCheck(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".pi"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("HOME", home)

	result := system.RunDoctor()

	foundDaemon := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "pi-sandboxd") {
			foundDaemon = true
			break
		}
	}
	if !foundDaemon {
		t.Errorf("RunDoctor did not check pi-sandboxd; issues: %v", result.Issues)
	}
}

// TestDoctor_HasRuntimeBackendInfo verifies RunDoctor reports runtime backends.
func TestDoctor_HasRuntimeBackendInfo(t *testing.T) {
	result := system.RunDoctor()

	found := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "runtime") || strings.Contains(iss.Message, "backend") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("RunDoctor did not report runtime backends; issues: %v", result.Issues)
	}
}
