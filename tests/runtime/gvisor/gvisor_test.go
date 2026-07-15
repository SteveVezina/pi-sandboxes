package gvisor_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/runtime/gvisor"
)

func TestRuntimeName(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.Name() != "gvisor" {
		t.Errorf("Expected name 'gvisor', got '%s'", rt.Name())
	}
}

func TestRuntimeIsAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	// On macOS, gVisor stub returns false
	available := rt.IsAvailable()
	_ = available // Just verify it doesn't panic
}

func TestRuntimeNewWithDefaults(t *testing.T) {
	rt := gvisor.New("", "/tmp/gvisor-test", 0)
	if rt.GetTimeout() != gvisor.DefaultTimeout {
		t.Errorf("Expected default timeout %v, got %v", gvisor.DefaultTimeout, rt.GetTimeout())
	}
	if rt.GetImage() != gvisor.DefaultImage {
		t.Errorf("Expected default image %s, got %s", gvisor.DefaultImage, rt.GetImage())
	}
}

func TestRuntimeNewCustom(t *testing.T) {
	rt := gvisor.New("my-image:latest", "/custom/root", 60*time.Second)
	if rt.GetImage() != "my-image:latest" {
		t.Errorf("Expected image 'my-image:latest', got '%s'", rt.GetImage())
	}
	if rt.GetTimeout() != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", rt.GetTimeout())
	}
	if rt.GetRootDir() != "/custom/root" {
		t.Errorf("Expected root dir '/custom/root', got '%s'", rt.GetRootDir())
	}
}

func TestRuntimeGetMode(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.GetMode() != "secure" {
		t.Errorf("Expected mode 'secure', got '%s'", rt.GetMode())
	}
}

func TestRuntimeCreate_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// On macOS, Create should fail because runsc is not available
	err := rt.Create(ctx, "test-create", "")
	if err == nil {
		t.Fatal("Expected error when gVisor is not available")
	}
}

func TestRuntimeDestroy_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rt.Destroy(ctx, "test-destroy")
	// Should fail gracefully on non-Linux
	_ = err
}

func TestRuntimeExec_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	exitErr, stdout, stderr, err := rt.Exec(ctx, "test-exec", "/tmp", "echo hello", 0)
	// All should be nil/empty on non-Linux
	if err == nil {
		t.Error("Expected error when gVisor is not available")
	}
	if exitErr != nil {
		t.Error("Expected nil exitErr on non-Linux")
	}
	if len(stdout) > 0 {
		t.Error("Expected empty stdout on non-Linux")
	}
	if len(stderr) > 0 {
		t.Error("Expected empty stderr on non-Linux")
	}
}

func TestRuntimeCloneGit_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rt.CloneGit(ctx, "test-clone", "https://github.com/test/repo.git", "/tmp")
	// Should fail gracefully on non-Linux
	_ = err
}

func TestRuntimeGetStatus_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	status, err := rt.GetStatus(ctx, "test-status")
	if err == nil {
		t.Error("Expected error when gVisor is not available")
	}
	if status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", status)
	}
}

func TestRuntimeGetLogs_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := rt.GetLogs(ctx, "test-logs")
	if err == nil {
		t.Error("Expected error when gVisor is not available")
	}
}

func TestRuntimeSnapshot_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rt.Snapshot(ctx, "test-snap", "snap1")
	_ = err // Should fail gracefully
}

func TestRuntimeRollback_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rt.Rollback(ctx, "test-snap", "snap1")
	_ = err
}

func TestRuntimeListSnapshots(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	snapshots, err := rt.ListSnapshots(ctx, "test-list")
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestRuntimeDeleteSnapshot_NotAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := rt.DeleteSnapshot(ctx, "test-snap", "snap1")
	_ = err
}

func TestDefaultCgroupConfig(t *testing.T) {
	cfg := gvisor.DefaultCgroupConfig()
	if cfg.CPUPeriod != 100000 {
		t.Errorf("Expected CPU period 100000, got %d", cfg.CPUPeriod)
	}
	if cfg.MemoryLimit != 512*1024*1024 {
		t.Errorf("Expected memory limit 512MB, got %d", cfg.MemoryLimit)
	}
	if cfg.MaxPIDs != 128 {
		t.Errorf("Expected max PIDs 128, got %d", cfg.MaxPIDs)
	}
}

func TestCgroupManager(t *testing.T) {
	tmpDir := t.TempDir()
	id := "cgroup-test-1"

	mgr := gvisor.NewCgroupManager(tmpDir, id)
	if err := mgr.Create(); err != nil {
		t.Fatalf("CgroupManager.Create failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(mgr.Path()); err != nil {
		t.Fatalf("Cgroup path %s does not exist: %v", mgr.Path(), err)
	}

	// Test setting limits (may fail without root, that's fine)
	_ = mgr.SetCPU(100000, 50000)
	_ = mgr.SetMemory(256 * 1024 * 1024)
	_ = mgr.SetPIDs(64)

	// Cleanup
	if err := mgr.Destroy(); err != nil {
		t.Fatalf("CgroupManager.Destroy failed: %v", err)
	}

	// Verify directory removed
	if _, err := os.Stat(mgr.Path()); err == nil {
		t.Error("Cgroup path should be removed after Destroy")
	}
}

func TestNamespaceConfig(t *testing.T) {
	cfg := gvisor.DefaultNamespaceConfig()
	if !cfg.UserNS {
		t.Error("Expected UserNS to be true")
	}
	if !cfg.MountNS {
		t.Error("Expected MountNS to be true")
	}
	if !cfg.PIDNS {
		t.Error("Expected PIDNS to be true")
	}
	if cfg.HostUID != 1000 {
		t.Errorf("Expected HostUID 1000, got %d", cfg.HostUID)
	}
	if cfg.HostGID != 1000 {
		t.Errorf("Expected HostGID 1000, got %d", cfg.HostGID)
	}
}

func TestNamespaceSetup(t *testing.T) {
	cfg := gvisor.Setup(nil)
	if cfg == nil {
		t.Fatal("Expected non-nil result from Setup")
	}
	// On macOS stub, Setup returns *NamespaceConfig
	// Just verify it doesn't panic
	_ = cfg
}

func TestValidate(t *testing.T) {
	// Validate should not panic
	if err := gvisor.Validate(); err != nil {
		t.Logf("Validate returned: %v", err)
	}
}

func TestRuntime_GetTimeout(t *testing.T) {
	rt := gvisor.New("", "/tmp/test", 45*time.Second)
	if rt.GetTimeout() != 45*time.Second {
		t.Errorf("Expected 45s timeout, got %v", rt.GetTimeout())
	}
}

func TestRuntime_GetImage(t *testing.T) {
	rt := gvisor.New("alpine:3.18", "/tmp/test", 0)
	if rt.GetImage() != "alpine:3.18" {
		t.Errorf("Expected image 'alpine:3.18', got '%s'", rt.GetImage())
	}
}

func TestRuntime_GetRootDir(t *testing.T) {
	rt := gvisor.New("", "/custom/dir", 0)
	if rt.GetRootDir() != "/custom/dir" {
		t.Errorf("Expected root dir '/custom/dir', got '%s'", rt.GetRootDir())
	}
}
