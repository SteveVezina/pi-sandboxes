package compat

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCreateDockerContainer_TimesOutWhenRuntimeStalls(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake runtime is POSIX-only")
	}

	dir := t.TempDir()
	fakeDocker := filepath.Join(dir, "docker")
	if err := os.WriteFile(fakeDocker, []byte("#!/bin/sh\nsleep 5\n"), 0755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ResetDetection()
	t.Cleanup(ResetDetection)

	previousTimeout := containerCommandTimeout
	containerCommandTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		containerCommandTimeout = previousTimeout
	})

	spec := &ContainerSpec{
		ID:          "timeout-test",
		Name:        "pi-sandbox-timeout-test",
		Image:       "debian:slim",
		NetworkMode: "bridge",
	}

	_, err := CreateContainer(spec)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
