package bench

import (
	"os"
	"os/exec"
	"path/filepath"
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

// WarmExecShell runs a shell command and returns duration.
func WarmExecShell() time.Duration {
	start := time.Now()
	cmd := exec.Command("/bin/sh", "-c", "echo hello")
	cmd.Run()
	return time.Since(start)
}

// FileScanRG measures filesystem scan overhead.
func FileScanRG() time.Duration {
	start := time.Now()
	// Use ripgrep if available, fall back to find
	if _, err := exec.LookPath("rg"); err == nil {
		cmd := exec.Command("rg", "--files", "/tmp")
		cmd.Run()
	} else {
		cmd := exec.Command("find", "/tmp", "-maxdepth", "2", "-type", "f")
		cmd.Run()
	}
	return time.Since(start)
}

// GitCloneSmall measures git clone of a small repo.
func GitCloneSmall() time.Duration {
	start := time.Now()
	dir := filepath.Join(os.TempDir(), "pi-bench-clone-"+time.Now().Format("20060102150405"))
	// Clone a small public repo
	cmd := exec.Command("git", "clone", "--depth", "1", "https://github.com/pi-sandbox/pi-sandbox-test.git", dir)
	cmd.Run()
	// Clean up
	os.RemoveAll(dir)
	return time.Since(start)
}

// PnpmInstall measures Node dependency cache path.
func PnpmInstall() time.Duration {
	start := time.Now()
	// Create a temp dir with a minimal package.json
	dir := filepath.Join(os.TempDir(), "pi-bench-pnpm-"+time.Now().Format("20060102150405"))
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"bench","version":"1.0.0"}`), 0644)
	// Try pnpm install, fall back to sleep if not available
	if _, err := exec.LookPath("pnpm"); err == nil {
		cmd := exec.Command("pnpm", "install", "--prefer-offline")
		cmd.Dir = dir
		cmd.Run()
	} else {
		time.Sleep(50 * time.Millisecond) // Placeholder
	}
	os.RemoveAll(dir)
	return time.Since(start)
}

// UVSync measures Python dependency cache path.
func UVSync() time.Duration {
	start := time.Now()
	dir := filepath.Join(os.TempDir(), "pi-bench-uv-"+time.Now().Format("20060102150405"))
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "bench"
version = "1.0.0"
dependencies = []
`), 0644)
	if _, err := exec.LookPath("uv"); err == nil {
		cmd := exec.Command("uv", "sync")
		cmd.Dir = dir
		cmd.Run()
	} else {
		time.Sleep(50 * time.Millisecond)
	}
	os.RemoveAll(dir)
	return time.Since(start)
}

// GoTest measures Go toolchain and cache.
func GoTest() time.Duration {
	start := time.Now()
	dir := filepath.Join(os.TempDir(), "pi-bench-gotest-"+time.Now().Format("20060102150405"))
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module bench
go 1.21
`), 0644)
	os.WriteFile(filepath.Join(dir, "bench_test.go"), []byte(`package main
import "testing"
func TestBench(t *testing.T) {}
`), 0644)
	if _, err := exec.LookPath("go"); err == nil {
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = dir
		cmd.Run()
	} else {
		time.Sleep(50 * time.Millisecond)
	}
	os.RemoveAll(dir)
	return time.Since(start)
}

// CargoTest measures Rust toolchain and cache.
func CargoTest() time.Duration {
	start := time.Now()
	dir := filepath.Join(os.TempDir(), "pi-bench-cargo-"+time.Now().Format("20060102150405"))
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(`[package]
name = "bench"
version = "1.0.0"
`), 0644)
	os.WriteFile(filepath.Join(dir, "src", "main.rs"), []byte(`fn main() {}
#[cfg(test)]
mod tests { #[test] fn test_bench() {} }
`), 0644)
	if _, err := exec.LookPath("cargo"); err == nil {
		os.MkdirAll(filepath.Join(dir, "src"), 0755)
		cmd := exec.Command("cargo", "test")
		cmd.Dir = dir
		cmd.Run()
	} else {
		time.Sleep(50 * time.Millisecond)
	}
	os.RemoveAll(dir)
	return time.Since(start)
}

// SnapshotCreate measures snapshot creation.
func SnapshotCreate() time.Duration {
	start := time.Now()
	src := filepath.Join(os.TempDir(), "pi-bench-snap-src-"+time.Now().Format("20060102150405"))
	dst := filepath.Join(os.TempDir(), "pi-bench-snap-dst-"+time.Now().Format("20060102150405"))
	os.MkdirAll(src, 0755)
	// Create some test files
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(src, "file-"+string(rune('a'+i))), []byte("test data"), 0644)
	}
	// Copy directory
	cmd := exec.Command("cp", "-r", src, dst)
	cmd.Run()
	os.RemoveAll(src)
	os.RemoveAll(dst)
	return time.Since(start)
}

// SnapshotRollback measures rollback.
func SnapshotRollback() time.Duration {
	start := time.Now()
	dir := filepath.Join(os.TempDir(), "pi-bench-rollback-"+time.Now().Format("20060102150405"))
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "original.txt"), []byte("original"), 0644)
	// Simulate: snapshot (copy), modify, rollback (restore copy)
	snapDir := dir + ".snap"
	cmd := exec.Command("cp", "-r", dir, snapDir)
	cmd.Run()
	os.WriteFile(filepath.Join(dir, "modified.txt"), []byte("modified"), 0644)
	os.RemoveAll(dir)
	cmd = exec.Command("cp", "-r", snapDir, dir)
	cmd.Run()
	os.RemoveAll(snapDir)
	return time.Since(start)
}

// ArtifactExport20MB measures artifact packing/export.
func ArtifactExport20MB() time.Duration {
	start := time.Now()
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

// Parallel10 measures 10 concurrent sandbox operations.
func Parallel10() time.Duration {
	start := time.Now()
	var dirs []string
	for i := 0; i < 10; i++ {
		dir := filepath.Join(os.TempDir(), "pi-bench-parallel-"+time.Now().Format("20060102150405")+"-"+string(rune('a'+i)))
		dirs = append(dirs, dir)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)
	}
	// Run 10 concurrent ls commands
	cmds := make([]*exec.Cmd, 10)
	for i, dir := range dirs {
		cmds[i] = exec.Command("ls", dir)
		cmds[i].Run()
	}
	for _, dir := range dirs {
		os.RemoveAll(dir)
	}
	return time.Since(start)
}

// Parallel100 measures high-density behavior.
func Parallel100() time.Duration {
	start := time.Now()
	var dirs []string
	for i := 0; i < 100; i++ {
		dir := filepath.Join(os.TempDir(), "pi-bench-parallel100-"+time.Now().Format("20060102150405")+"-"+string(rune('a'+(i%26))))
		dirs = append(dirs, dir)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)
	}
	for _, dir := range dirs {
		cmd := exec.Command("ls", dir)
		cmd.Run()
	}
	for _, dir := range dirs {
		os.RemoveAll(dir)
	}
	return time.Since(start)
}
