package oci_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
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
	if strings.Contains(recorded, "--rm") {
		t.Errorf("create args must not include --rm; daemon destroy/reconciliation owns cleanup, got: %s", recorded)
	}
}

func TestCLIEngine_Create_WorkspaceExecAllowed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		Name:      "pi-sandbox-exec",
		Image:     "debian:bookworm-slim",
		Workspace: "/tmp/ws",
		Artifacts: "/tmp/art",
		Caches:    map[string]string{"pnpm": "/tmp/cache-pnpm"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	args, _ := os.ReadFile(argsFile)
	recorded := string(args)

	// Workspace-class mounts must allow exec (./gradlew, node_modules/.bin,
	// .venv/bin) — noexec only on /tmp and secret mounts (SPEC §14.7.5).
	for _, mount := range []string{"/tmp/ws:/workspace:", "/tmp/art:/artifacts:", "/tmp/cache-pnpm:/cache/pnpm:"} {
		idx := strings.Index(recorded, mount)
		if idx < 0 {
			t.Fatalf("mount %q missing in args: %s", mount, recorded)
		}
		opts := recorded[idx : idx+len(mount)+24]
		if strings.Contains(opts, "noexec") {
			t.Errorf("workspace-class mount %q must not be noexec, got %q", mount, opts)
		}
	}
	if !strings.Contains(recorded, "/tmp:rw,nosuid,noexec") {
		t.Errorf("/tmp tmpfs must stay noexec, got: %s", recorded)
	}
	if strings.Contains(recorded, "/home/agent:rw,nosuid,noexec") {
		t.Errorf("/home/agent must allow exec (user-installed tools), got: %s", recorded)
	}
}

func TestCLIEngine_Create_UsesNamedVolumesWithoutHostWorkspaceBind(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		Name:      "pi-sandbox-managed",
		Image:     "debian:bookworm-slim",
		Workspace: "pi-sandbox-s1-workspace",
		Artifacts: "pi-sandbox-s1-artifacts",
		Caches:    map[string]string{"npm": "pi-sandbox-s1-cache-npm"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	args, _ := os.ReadFile(argsFile)
	recorded := string(args)
	for _, want := range []string{
		"-v pi-sandbox-s1-workspace:/workspace:",
		"-v pi-sandbox-s1-artifacts:/artifacts:",
		"-v pi-sandbox-s1-cache-npm:/cache/npm:",
	} {
		if !strings.Contains(recorded, want) {
			t.Errorf("create args missing managed volume %q; got: %s", want, recorded)
		}
	}
	if strings.Contains(recorded, "-v /workspace:/workspace:") {
		t.Fatalf("default compat creation must not bind host /workspace; got: %s", recorded)
	}
}

func TestCLIEngine_Create_EgressNoneUsesNetworkNone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		SandboxID: "s1", Name: "pi-sandbox-s1", Image: "debian:bookworm-slim",
		Network: pruntime.NetworkSpec{Mode: "none"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recorded := readFile(t, argsFile)
	if !strings.Contains(recorded, "--network none") {
		t.Errorf("none mode must use --network none; got: %s", recorded)
	}
	if strings.Contains(recorded, "HTTP_PROXY") {
		t.Errorf("none mode must not inject proxy env; got: %s", recorded)
	}
}

func TestCLIEngine_Create_EgressRestrictedInjectsSandboxScopedProxy(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		SandboxID: "sbx-42", Name: "pi-sandbox-sbx-42", Image: "debian:bookworm-slim",
		Network: pruntime.NetworkSpec{Mode: "restricted", ProxyAddr: "127.0.0.1:9002"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recorded := readFile(t, argsFile)
	for _, want := range []string{
		"--network bridge",
		"-e HTTP_PROXY=http://sbx-42:x@127.0.0.1:9002",
		"-e HTTPS_PROXY=http://sbx-42:x@127.0.0.1:9002",
		"-e NO_PROXY=localhost,127.0.0.1,::1",
	} {
		if !strings.Contains(recorded, want) {
			t.Errorf("restricted mode missing %q; got: %s", want, recorded)
		}
	}
}

func TestCLIEngine_Create_EgressRestrictedWithoutProxyAddrIsBridgeOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		SandboxID: "s1", Name: "pi-sandbox-s1", Image: "debian:bookworm-slim",
		Network: pruntime.NetworkSpec{Mode: "restricted"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	recorded := readFile(t, argsFile)
	if strings.Contains(recorded, "HTTP_PROXY") {
		t.Errorf("no ProxyAddr → no proxy env; got: %s", recorded)
	}
	if !strings.Contains(recorded, "--network bridge") {
		t.Errorf("want --network bridge; got: %s", recorded)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func TestCLIEngine_Create_AppliesResourceLimits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	_, err := eng.Create(context.Background(), &oci.ContainerSpec{
		Name:  "pi-sandbox-limits",
		Image: "debian:bookworm-slim",
		Limits: pruntime.ResourceLimits{
			MemoryBytes: 2 << 30,
			CPUs:        2,
			PIDs:        256,
			OpenFiles:   1024,
		},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	args, _ := os.ReadFile(argsFile)
	recorded := string(args)
	for _, want := range []string{"--memory 2147483648", "--cpus 2", "--pids-limit 256", "--ulimit nofile=1024:1024"} {
		if !strings.Contains(recorded, want) {
			t.Errorf("create args missing resource limit %q; got: %s", want, recorded)
		}
	}
}

func TestDockerEngine_RunsAsUnprivilegedUser(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	if _, err := eng.Create(context.Background(), &oci.ContainerSpec{Name: "n", Image: "i"}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	args, _ := os.ReadFile(argsFile)
	if !strings.Contains(string(args), "--user 1000:1000") {
		t.Errorf("docker containers must run as explicit unprivileged user, got: %s", args)
	}
}

func TestPodmanEngine_KeepsUserID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewPodmanEngine(binary)

	if _, err := eng.Create(context.Background(), &oci.ContainerSpec{Name: "n", Image: "i"}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	args, _ := os.ReadFile(argsFile)
	recorded := string(args)
	if !strings.Contains(recorded, "--userns=keep-id") || !strings.Contains(recorded, "--user ") {
		t.Errorf("podman containers must map to the invoking user, got: %s", recorded)
	}
}

func TestDockerEngine_PassesVersionedSeccompProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	binary, argsFile := fakeCLI(t, "echo abc\n")
	eng := oci.NewDockerEngine(binary)

	if _, err := eng.Create(context.Background(), &oci.ContainerSpec{Name: "n", Image: "i"}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	args, _ := os.ReadFile(argsFile)
	recorded := string(args)

	marker := "seccomp="
	idx := strings.Index(recorded, marker)
	if idx < 0 {
		t.Fatalf("expected explicit seccomp profile in args, got: %s", recorded)
	}
	path := strings.Fields(recorded[idx+len(marker):])[0]
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("seccomp profile not written: %v", err)
	}
	if !strings.Contains(string(data), "defaultAction") {
		t.Errorf("seccomp profile malformed: %s", data)
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
