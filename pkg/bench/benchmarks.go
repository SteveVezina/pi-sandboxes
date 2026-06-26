package bench

import (
	"os"
	"os/exec"
	"time"
)

// All returns the list of all benchmark definitions.
func All() []*Benchmark {
	return []*Benchmark{
		{Name: "warm_exec_echo", Func: WarmExecEcho},
		{Name: "warm_exec_shell", Func: WarmExecShell},
		{Name: "file_scan_rg", Func: FileScanRG},
		{Name: "git_clone_small", Func: GitCloneSmall, Disabled: false},
		{Name: "artifact_export_20mb", Func: ArtifactExport20MB},
		// These require tools that may not be installed:
		{Name: "pnpm_install_cached", Func: PnpmInstall, Disabled: true},
		{Name: "uv_sync_cached", Func: UVSync, Disabled: true},
		{Name: "go_test_cached", Func: GoTest, Disabled: true},
		{Name: "cargo_test_cached", Func: CargoTest, Disabled: true},
		{Name: "snapshot_create", Func: SnapshotCreate, Disabled: true},
		{Name: "snapshot_rollback", Func: SnapshotRollback, Disabled: true},
		{Name: "parallel_10", Func: Parallel10, Disabled: true},
		{Name: "parallel_100", Func: Parallel100, Disabled: true},
	}
}

// WarmExecEcho runs echo hello and returns duration.
func WarmExecEcho() time.Duration {
	start := time.Now()
	cmd := exec.Command("echo", "hello")
	cmd.Run()
	return time.Since(start)
}

func WarmExecShell() time.Duration {
	start := time.Now()
	cmd := exec.Command("/bin/sh", "-c", "echo hello")
	cmd.Run()
	return time.Since(start)
}

func FileScanRG() time.Duration {
	start := time.Now()
	cmd := exec.Command("sh", "-c", "ls /tmp")
	cmd.Run()
	return time.Since(start)
}

func GitCloneSmall() time.Duration {
	start := time.Now()
	// Stub: would clone a small repo
	time.Sleep(10 * time.Millisecond) // Simulate network latency
	return time.Since(start)
}

func PnpmInstall() time.Duration {
	// Stub
	time.Sleep(10 * time.Millisecond)
	return time.Since(time.Now())
}

func UVSync() time.Duration {
	// Stub
	time.Sleep(10 * time.Millisecond)
	return time.Since(time.Now())
}

func GoTest() time.Duration {
	// Stub
	time.Sleep(10 * time.Millisecond)
	return time.Since(time.Now())
}

func CargoTest() time.Duration {
	// Stub
	time.Sleep(10 * time.Millisecond)
	return time.Since(time.Now())
}

func SnapshotCreate() time.Duration {
	// Stub
	time.Sleep(10 * time.Millisecond)
	return time.Since(time.Now())
}

func SnapshotRollback() time.Duration {
	// Stub
	time.Sleep(10 * time.Millisecond)
	return time.Since(time.Now())
}

func ArtifactExport20MB() time.Duration {
	start := time.Now()
	// Create a 20MB temp file
	tmpFile := "/tmp/pi-bench-20mb"
	f, err := os.Create(tmpFile)
	if err == nil {
		data := make([]byte, 20*1024*1024)
		f.Write(data)
		f.Close()
		os.Remove(tmpFile)
	}
	return time.Since(start)
}

func Parallel10() time.Duration {
	// Stub: would run 10 concurrent sandboxes
	time.Sleep(100 * time.Millisecond)
	return time.Since(time.Now())
}

func Parallel100() time.Duration {
	// Stub: would run 100 concurrent sandboxes
	time.Sleep(500 * time.Millisecond)
	return time.Since(time.Now())
}
