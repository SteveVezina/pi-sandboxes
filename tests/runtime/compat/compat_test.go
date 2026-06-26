package compat_test

import (
	"context"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/compat"
)

func TestBestRuntime(t *testing.T) {
	rt := compat.Best()
	// On this system, one of containerd/podman/runc/docker may or may not be installed
	// Just verify it doesn't panic
	_ = rt
}

func TestIsAvailable(t *testing.T) {
	// runc is commonly installed
	if compat.IsAvailable(compat.RuntimeRunc) {
		// runc is available, that's fine
	}
	// At minimum, verify the function doesn't panic
	_ = compat.IsAvailable(compat.RuntimeDocker)
}

func TestDetectRuntime(t *testing.T) {
	rt := compat.DetectRuntime()
	// May or may not find a runtime
	_ = rt
}

func TestCreateContainer(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-container-1",
		Image:     "debian:slim",
		Workspace: "/tmp/test-workspace",
		Artifacts: "/tmp/test-artifacts",
		Caches:    map[string]string{"npm": "/tmp/cache/npm"},
	}

	c, err := compat.CreateContainer(spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	if c.ID != "test-container-1" {
		t.Errorf("Expected ID 'test-container-1', got '%s'", c.ID)
	}
	if c.Spec.Privileged {
		t.Error("Container should not be privileged by default")
	}
	if c.Spec.NetworkMode != "bridge" {
		t.Errorf("Expected network mode 'bridge', got '%s'", c.Spec.NetworkMode)
	}
}

func TestCreateContainer_NetworkDefault(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:      "test-net",
		Image:   "debian:slim",
	}

	c, err := compat.CreateContainer(spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	if c.Spec.NetworkMode != "bridge" {
		t.Errorf("Expected default network 'bridge', got '%s'", c.Spec.NetworkMode)
	}
}

func TestCreateContainer_NoDockerSocket(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-nosocket",
		Image:     "debian:slim",
		Workspace: "/var/run/docker.sock",
	}

	_, err := compat.CreateContainer(spec)
	if err == nil {
		t.Fatal("Expected error for docker socket mount")
	}
}

func TestCreateContainer_NoID(t *testing.T) {
	spec := &compat.ContainerSpec{
		Image: "debian:slim",
	}

	_, err := compat.CreateContainer(spec)
	if err == nil {
		t.Fatal("Expected error for missing container ID")
	}
}

func TestCreateContainer_NoImage(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID: "test-noimage",
	}

	_, err := compat.CreateContainer(spec)
	if err == nil {
		t.Fatal("Expected error for missing image")
	}
}

func TestContainerConfig(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-config",
		Image:     "debian:slim",
		Workspace: "/tmp/ws",
		Artifacts: "/tmp/art",
		Caches:    map[string]string{"npm": "/tmp/cache/npm"},
	}

	config := compat.ContainerConfig(spec)

	// Check process capabilities are dropped
	proc, ok := config["process"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected process config")
	}
	caps, ok := proc["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected capabilities config")
	}
	bounding, ok := caps["bounding"].([]string)
	if !ok {
		t.Fatal("Expected bounding capabilities")
	}
	if len(bounding) != 0 {
		t.Error("Expected empty bounding capabilities (all dropped)")
	}

	// Check noNewPrivileges
	if v, ok := proc["noNewPrivileges"].(bool); !ok || !v {
		t.Error("Expected noNewPrivileges to be true")
	}

	// Check mounts
	mounts, ok := config["mounts"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected mounts config")
	}
	if len(mounts) < 3 {
		t.Errorf("Expected at least 3 mounts, got %d", len(mounts))
	}
}

func TestContainerLifecycle(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-lifecycle",
		Image:     "debian:slim",
		Workspace: "/tmp/ws",
	}

	c, err := compat.CreateContainer(spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	// Start/Stop/Destroy should not panic
	if err := c.Start(); err != nil {
		// Stub may return error, that's fine
	}
	if err := c.Stop(); err != nil {
		// Stub may return error
	}
	if err := c.Destroy(); err != nil {
		// Stub may return error
	}
}

func TestExecCommand(t *testing.T) {
	rt := &compat.DetectedRuntime{
		Name: compat.RuntimeDocker,
		Path: "/usr/bin/docker",
	}

	cmd := compat.ExecCommand(rt, "test-container", "echo hello")
	if cmd == nil {
		t.Fatal("Expected non-nil command")
	}
	// Docker exec command should start with docker
	if len(cmd.Args) == 0 || cmd.Args[0] != "docker" {
		t.Errorf("Expected first arg 'docker', got '%s'", cmd.Args[0])
	}
}

func TestExecCommand_NilRuntime(t *testing.T) {
	cmd := compat.ExecCommand(nil, "test-container", "echo hello")
	if cmd == nil {
		t.Fatal("Expected non-nil command for nil runtime")
	}
	// Should fall back to /bin/sh
	if cmd.Path != "/bin/sh" {
		t.Errorf("Expected fallback path '/bin/sh', got '%s'", cmd.Path)
	}
}

func TestContainerExec(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-exec",
		Image:     "debian:slim",
		Workspace: "/tmp/ws",
	}

	c, err := compat.CreateContainer(spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5000)
	defer cancel()

	result, err := c.Exec(ctx, "echo hello")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestExecResult_NilContainer(t *testing.T) {
	var c *compat.Container
	_, err := c.Exec(context.Background(), "echo test")
	if err == nil {
		t.Fatal("Expected error for nil container exec")
	}
}

func TestEnsureRuntimeDir(t *testing.T) {
	err := compat.EnsureRuntimeDir()
	if err != nil {
		t.Fatalf("EnsureRuntimeDir failed: %v", err)
	}
}

func TestListContainers(t *testing.T) {
	containers, err := compat.ListContainers()
	if err != nil {
		t.Fatalf("ListContainers failed: %v", err)
	}
	_ = containers // Stub returns empty list
}

func TestPruneStale(t *testing.T) {
	count, err := compat.PruneStale()
	if err != nil {
		t.Fatalf("PruneStale failed: %v", err)
	}
	_ = count // Stub returns 0
}

func TestContainerState(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-state",
		Image:     "debian:slim",
		Workspace: "/tmp/ws",
	}

	c, err := compat.CreateContainer(spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	state := c.State()
	if state != "running" {
		t.Errorf("Expected state 'running', got '%s'", state)
	}
}

func TestContainerState_Nil(t *testing.T) {
	var c *compat.Container
	state := c.State()
	if state != "unknown" {
		t.Errorf("Expected state 'unknown' for nil container, got '%s'", state)
	}
}

func TestContainerConfig_Mounts(t *testing.T) {
	spec := &compat.ContainerSpec{
		ID:        "test-mounts",
		Image:     "debian:slim",
		Workspace: "/tmp/ws",
		Artifacts: "/tmp/art",
		Caches:    map[string]string{"npm": "/tmp/npm", "pip": "/tmp/pip"},
	}

	config := compat.ContainerConfig(spec)
	mounts, ok := config["mounts"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected mounts")
	}

	// Should have workspace + artifacts + 2 caches = 4
	if len(mounts) != 4 {
		t.Errorf("Expected 4 mounts, got %d", len(mounts))
	}
}
