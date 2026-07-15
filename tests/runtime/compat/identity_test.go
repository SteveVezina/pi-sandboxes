package compat_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/compat"
)

// TestCreateContainer_PreservesSandboxID verifies the PROP-008 identity
// rule: the stable sandbox ID is never overwritten with the runtime
// container ID (old code mutated spec.ID after create).
func TestCreateContainer_PreservesSandboxID(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "docker")
	script := "#!/bin/sh\necho fedcba9876543210aaaa\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	t.Setenv("PATH", dir)
	compat.ResetDetection()
	t.Cleanup(compat.ResetDetection)

	spec := &compat.ContainerSpec{
		ID:    "sandbox-1234-uuid",
		Image: "debian:bookworm-slim",
	}
	container, err := compat.CreateContainer(spec)
	if err != nil {
		t.Fatalf("CreateContainer failed: %v", err)
	}

	if spec.ID != "sandbox-1234-uuid" {
		t.Errorf("sandbox ID mutated to %q — must stay stable", spec.ID)
	}
	if container.ID != "sandbox-1234-uuid" {
		t.Errorf("container sandbox ID = %q, want sandbox-1234-uuid", container.ID)
	}
	if container.RuntimeObjectID != "fedcba987654" {
		t.Errorf("RuntimeObjectID = %q, want truncated runtime container ID", container.RuntimeObjectID)
	}
}
