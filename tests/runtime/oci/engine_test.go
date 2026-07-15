package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/runtime/oci"
)

// fakeCLI writes a fake docker/podman binary that records its arguments
// and prints a canned container ID.
func fakeCLI(t *testing.T, script string) (binary, argsFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args.txt")
	binary = filepath.Join(dir, "fake-oci")
	full := "#!/bin/sh\necho \"$@\" >> " + argsFile + "\n" + script
	if err := os.WriteFile(binary, []byte(full), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	return binary, argsFile
}

func TestCLIEngine_Create_ReturnsContainerID(t *testing.T) {
	binary, argsFile := fakeCLI(t, "echo 0123456789abcdef0123\n")
	eng := oci.NewDockerEngine(binary)

	id, err := eng.Create(context.Background(), &oci.ContainerSpec{
		Name:      "pi-sandbox-test",
		Image:     "debian:bookworm-slim",
		Workspace: "/tmp/ws",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id != "0123456789ab" {
		t.Errorf("expected truncated container ID, got %q", id)
	}

	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	recorded := string(args)
	for _, want := range []string{"--cap-drop ALL", "--security-opt no-new-privileges", "--read-only", "--label pi-sandbox=true", "-v /tmp/ws:/workspace:"} {
		if !strings.Contains(recorded, want) {
			t.Errorf("create args missing hardened default %q; got: %s", want, recorded)
		}
	}
}

func TestCLIEngine_Create_TimesOutOnStalledRuntime(t *testing.T) {
	binary, _ := fakeCLI(t, "sleep 10\n")
	eng := oci.NewDockerEngine(binary)
	eng.Timeout = 200 * time.Millisecond

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		Name:  "pi-sandbox-stall",
		Image: "debian:bookworm-slim",
	})
	if err == nil {
		t.Fatal("expected stalled runtime creation to fail, not hang")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestCLIEngine_Create_RequiresNameAndImage(t *testing.T) {
	eng := oci.NewDockerEngine("/nonexistent")
	if _, err := eng.Create(context.Background(), &oci.ContainerSpec{Image: "x"}); err == nil {
		t.Error("expected missing name to fail")
	}
	if _, err := eng.Create(context.Background(), &oci.ContainerSpec{Name: "x"}); err == nil {
		t.Error("expected missing image to fail")
	}
}

func TestCLIEngine_Inspect_ParsesStatus(t *testing.T) {
	binary, _ := fakeCLI(t, "echo running\n")
	eng := oci.NewPodmanEngine(binary)

	status, err := eng.Inspect(context.Background(), "pi-sandbox-test")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}
	if status != "running" {
		t.Errorf("expected running, got %q", status)
	}
}

func TestCLIEngine_List_ParsesContainers(t *testing.T) {
	binary, _ := fakeCLI(t, `printf 'abc123|pi-sandbox-a|Up 2 minutes|debian:slim\ndef456|pi-sandbox-b|Exited (0)|node:22\n'`+"\n")
	eng := oci.NewDockerEngine(binary)

	list, err := eng.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(list))
	}
	if !list[0].Running || list[1].Running {
		t.Errorf("running flags wrong: %+v", list)
	}
}
