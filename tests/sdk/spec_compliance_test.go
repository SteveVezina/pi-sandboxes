package sdk_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func specTestRepoRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("cannot find repo root")
	return ""
}

func buildSpecTestBin(t *testing.T) string {
	t.Helper()
	root := specTestRepoRoot(t)
	out := filepath.Join(t.TempDir(), "pi")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/pi-box")
	cmd.Dir = root
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build pi-box: %v\n%s", err, b)
	}
	return out
}

// TestPythonSDKStreamingInterface verifies AC-15: the Python SDK supports
// streaming output via exec_stream / ExecStreamEvent.
func TestPythonSDKStreamingInterface(t *testing.T) {
	root := specTestRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "sdk", "python", "src", "pi_sandbox", "__init__.py"))
	if err != nil {
		t.Fatalf("read python sdk: %v", err)
	}
	code := string(src)
	for _, want := range []string{"exec_stream", "ExecStreamEvent", "Generator", "event_type"} {
		if !strings.Contains(code, want) {
			t.Errorf("python SDK missing streaming symbol %q (AC-15)", want)
		}
	}
}

// TestTypeScriptSDKStreamingInterface verifies AC-15: the TypeScript SDK
// supports streaming output via execStream and ExecStreamEvent.
func TestTypeScriptSDKStreamingInterface(t *testing.T) {
	root := specTestRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "sdk", "typescript", "src", "client.ts"))
	if err != nil {
		t.Fatalf("read typescript sdk: %v", err)
	}
	code := string(src)
	for _, want := range []string{"execStream", "ExecStreamEvent", "AsyncGenerator"} {
		if !strings.Contains(code, want) {
			t.Errorf("typescript SDK missing streaming symbol %q (AC-15)", want)
		}
	}
}

// TestPythonSDKExecStreamCallsExecEndpoint verifies exec_stream targets the
// /exec endpoint and yields ExecStreamEvent objects (AC-15 / §21).
func TestPythonSDKExecStreamCallsExecEndpoint(t *testing.T) {
	root := specTestRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "sdk", "python", "src", "pi_sandbox", "__init__.py"))
	if err != nil {
		t.Fatalf("read sdk: %v", err)
	}
	code := string(src)
	if !strings.Contains(code, "/exec") {
		t.Error("exec_stream does not call /exec endpoint (AC-15)")
	}
	if !strings.Contains(code, "yield ExecStreamEvent") {
		t.Error("exec_stream does not yield ExecStreamEvent objects (AC-15)")
	}
}

// TestTypeScriptSDKExecStreamIsGenerator verifies TypeScript execStream is an
// AsyncGenerator that calls /exec (AC-15 / §21).
func TestTypeScriptSDKExecStreamIsGenerator(t *testing.T) {
	root := specTestRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "sdk", "typescript", "src", "client.ts"))
	if err != nil {
		t.Fatalf("read sdk: %v", err)
	}
	code := string(src)
	if !strings.Contains(code, "/exec") {
		t.Error("execStream does not call /exec endpoint (AC-15)")
	}
	if !strings.Contains(code, "yield") {
		t.Error("execStream is not a generator (no yield) (AC-15)")
	}
}

// TestDaemonRouterHasAllSPECRoutes verifies SPEC.md §9 required routes are
// registered in pkg/daemon/router.go (contract consistency check).
func TestDaemonRouterHasAllSPECRoutes(t *testing.T) {
	root := specTestRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "pkg", "daemon", "router.go"))
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	code := string(src)
	required := []string{
		`"/v1/sandboxes"`,
		`"/v1/sandboxes/{id}"`,
		`"/v1/sandboxes/{id}/clone"`,
		`"/v1/sandboxes/{id}/exec"`,
		`"/v1/sandboxes/{id}/files/write"`,
		`"/v1/sandboxes/{id}/files/read"`,
		`"/v1/sandboxes/{id}/files/pull"`,
		`"/v1/sandboxes/{id}/files/push"`,
		`"/v1/sandboxes/{id}/diff"`,
		`"/v1/sandboxes/{id}/patch"`,
		`"/v1/sandboxes/{id}/output"`,
		`"/v1/sandboxes/{id}/snapshot"`,
		`"/v1/sandboxes/{id}/rollback"`, // SPEC §9 canonical rollback path
		`"/v1/sandboxes/{id}/logs"`,
	}
	for _, r := range required {
		if !strings.Contains(code, r) {
			t.Errorf("router.go missing required SPEC.md §9 route %s", r)
		}
	}
}

// TestBenchmarkAllThirteenRegistered verifies AC-14: all 13 SPEC-required
// benchmark names are registered in pkg/bench/benchmarks.go.
func TestBenchmarkAllThirteenRegistered(t *testing.T) {
	root := specTestRepoRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "pkg", "bench", "benchmarks.go"))
	if err != nil {
		t.Fatalf("read benchmarks.go: %v", err)
	}
	code := string(src)
	required := []string{
		`"warm_exec_echo"`, `"warm_exec_shell"`, `"file_scan_rg"`, `"git_clone_small"`,
		`"pnpm_install_cached"`, `"uv_sync_cached"`, `"go_test_cached"`, `"cargo_test_cached"`,
		`"snapshot_create"`, `"snapshot_rollback"`,
		`"artifact_export_20mb"`, `"parallel_10"`, `"parallel_100"`,
	}
	for _, name := range required {
		if !strings.Contains(code, name) {
			t.Errorf("benchmarks.go missing required benchmark %s (SPEC.md AC-14)", name)
		}
	}
}

// TestSDKCreateCloneExecDiffMethods verifies AC-15: both SDKs expose the
// four required sandbox operation methods.
func TestSDKCreateCloneExecDiffMethods(t *testing.T) {
	root := specTestRepoRoot(t)
	checks := []struct {
		path    string
		methods []string
	}{
		{
			filepath.Join(root, "sdk", "typescript", "src", "client.ts"),
			[]string{"create(", "clone(", "exec(", "diff("},
		},
		{
			filepath.Join(root, "sdk", "python", "src", "pi_sandbox", "__init__.py"),
			[]string{"def create", "def clone", "def exec", "def diff"},
		},
	}
	for _, tc := range checks {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		for _, m := range tc.methods {
			if !strings.Contains(string(data), m) {
				t.Errorf("%s missing method %q (AC-15)", tc.path, m)
			}
		}
	}
}

// TestDestroyAllFlagDocumented verifies AC-8.4: pi-box box destroy --help
// documents the --all flag.
func TestDestroyAllFlagDocumented(t *testing.T) {
	bin := buildSpecTestBin(t)
	out, _ := exec.Command(bin, "box", "destroy", "--help").CombinedOutput()
	if !strings.Contains(string(out), "--all") {
		t.Errorf("pi-box box destroy --help does not document --all (AC-8.4):\n%s", out)
	}
}

// TestJSONFlagDocumented verifies AC-1.5: pi-box box --help mentions --json flag.
func TestJSONFlagDocumented(t *testing.T) {
	bin := buildSpecTestBin(t)
	out, _ := exec.Command(bin, "box", "--help").CombinedOutput()
	if !strings.Contains(string(out), "--json") {
		t.Errorf("pi-box box --help does not document --json flag (AC-1.5):\n%s", out)
	}
}
