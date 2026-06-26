package fast_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/fast"
)

func TestDefaultCgroupConfig(t *testing.T) {
	cfg := fast.DefaultCgroupConfig()

	if cfg.CPUPeriod != 100000 {
		t.Errorf("Expected CPUPeriod 100000, got %d", cfg.CPUPeriod)
	}
	if cfg.MaxPIDs != 256 {
		t.Errorf("Expected MaxPIDs 256, got %d", cfg.MaxPIDs)
	}
}

func TestCgroupManager_Path(t *testing.T) {
	base := filepath.Join(os.TempDir(), "pi-cgroup-test")
	os.RemoveAll(base)
	defer os.RemoveAll(base)

	m := fast.NewCgroupManager(base, "test-sandbox")
	expected := filepath.Join(base, "test-sandbox")
	if m.Path() != expected {
		t.Errorf("Expected path %s, got %s", expected, m.Path())
	}
}

func TestCgroupCreateFailsOnNonLinux(t *testing.T) {
	m := fast.NewCgroupManager(os.TempDir(), "test")
	err := m.Create()
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestCgroupSetCPUFailsOnNonLinux(t *testing.T) {
	m := fast.NewCgroupManager(os.TempDir(), "test")
	err := m.SetCPU(100000, 50000)
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestCgroupSetMemoryFailsOnNonLinux(t *testing.T) {
	m := fast.NewCgroupManager(os.TempDir(), "test")
	err := m.SetMemory(1024 * 1024)
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestCgroupSetPIDsFailsOnNonLinux(t *testing.T) {
	m := fast.NewCgroupManager(os.TempDir(), "test")
	err := m.SetPIDs(256)
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestCgroupAddProcessFailsOnNonLinux(t *testing.T) {
	m := fast.NewCgroupManager(os.TempDir(), "test")
	err := m.AddProcess(1234)
	if err == nil {
		t.Fatal("Expected error on non-Linux, got nil")
	}
}

func TestCgroupDestroy(t *testing.T) {
	// Destroy should not error even on non-Linux (stub cleans up)
	m := fast.NewCgroupManager(os.TempDir(), "test")
	err := m.Destroy()
	if err != nil {
		t.Errorf("Destroy should not error, got: %v", err)
	}
}
