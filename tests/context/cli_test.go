package context_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildCLIBinary builds the pi binary into a temp dir and returns its path.
func buildCLIBinary(t *testing.T) string {
	t.Helper()
	root, err := repoRoot()
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	out := filepath.Join(t.TempDir(), "pi")
	if runtime.GOOS == "windows" {
		out += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", out, "./cmd/pi")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build cmd/pi: %v\n%s", err, output)
	}
	return out
}

func repoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", os.ErrNotExist
}

// TestCLI_ContextCreateUseList verifies the pi context CLI group works
// end-to-end with a temp contexts.yaml (F22 / AC-25.1, 25.2, 25.3, 25.6).
func TestCLI_ContextCreateUseList(t *testing.T) {
	bin := buildCLIBinary(t)
	store := filepath.Join(t.TempDir(), "contexts.yaml")

	run := func(args ...string) (string, error) {
		full := append([]string{"context", "--store", store}, args...)
		cmd := exec.Command(bin, full...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// Create a remote http context.
	out, err := run("create", "ws", "https://daemon:7777", "--token-env", "PI_TOKEN_WS")
	if err != nil {
		t.Fatalf("context create: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Created context") {
		t.Fatalf("create output = %q", out)
	}

	// Use it.
	out, err = run("use", "ws")
	if err != nil {
		t.Fatalf("context use: %v\n%s", err, out)
	}

	// List shows it as active.
	out, err = run("list")
	if err != nil {
		t.Fatalf("context list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ws") {
		t.Fatalf("list output missing 'ws': %q", out)
	}
	// Active context marker should appear next to ws.
	lines := strings.Split(out, "\n")
	foundActive := false
	for _, line := range lines {
		if strings.Contains(line, " ws ") && strings.HasPrefix(strings.TrimSpace(line), "*") {
			foundActive = true
			break
		}
	}
	if !foundActive {
		t.Fatalf("active marker missing on ws line:\n%s", out)
	}

	// Inspect returns ws details (AC-25.7).
	out, err = run("inspect", "ws")
	if err != nil {
		t.Fatalf("context inspect: %v\n%s", err, out)
	}
	for _, want := range []string{"name:", "target:", "transport:", "auth.type:"} {
		if !strings.Contains(out, want) {
			t.Errorf("inspect missing %q:\n%s", want, out)
		}
	}
}

// TestCLI_ContextOverrideFlagPresent verifies AC-25.5/25.8 — the global
// --context flag exists on the root command.
func TestCLI_ContextOverrideFlagPresent(t *testing.T) {
	bin := buildCLIBinary(t)
	cmd := exec.Command(bin, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pi --help: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "--context") {
		t.Fatalf("pi --help does not document --context override:\n%s", out)
	}
}

// TestCLI_RemoteContextOverrideRoutesToRemote verifies that --context <name>
// causes `pi box list` to talk to the remote daemon target (AC-25.5/25.8 +
// AC-25.4 routing). Uses an httptest server as the remote target.
func TestCLI_RemoteContextOverrideRoutesToRemote(t *testing.T) {
	bin := buildCLIBinary(t)

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer hello-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"remote-1","name":"remote-sb"}]`))
	}))
	defer srv.Close()

	store := filepath.Join(t.TempDir(), "contexts.yaml")
	// Set up a context via the CLI.
	setup := exec.Command(bin, "context", "--store", store, "create", "ws",
		srv.URL, "--token-env", "PI_OVERRIDE_TOKEN")
	if out, err := setup.CombinedOutput(); err != nil {
		t.Fatalf("setup context: %v\n%s", err, out)
	}

	// Now run `pi --context ws box list` and check the remote was hit.
	// We must point the store at our temp file via PI_CONTEXTS_PATH if the
	// box command uses DefaultPath. Since we cannot inject the store path
	// into box subcommands today, copy the temp store to ~/.pi/contexts.yaml
	// for the duration of this test.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_OVERRIDE_TOKEN", "hello-token")
	if err := os.MkdirAll(filepath.Join(home, ".pi"), 0o755); err != nil {
		t.Fatalf("mkdir ~/.pi: %v", err)
	}
	// Copy the temp store to the test HOME.
	data, err := os.ReadFile(store)
	if err != nil {
		t.Fatalf("read temp store: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".pi", "contexts.yaml"), data, 0o600); err != nil {
		t.Fatalf("write ~/.pi/contexts.yaml: %v", err)
	}

	cmd := exec.Command(bin, "--context", "ws", "box", "list")
	cmd.Env = append(os.Environ(), "HOME="+home, "PI_OVERRIDE_TOKEN=hello-token")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pi --context ws box list: %v\n%s", err, out)
	}
	if hits == 0 {
		t.Fatalf("remote daemon was never hit; output:\n%s", out)
	}
	if !strings.Contains(string(out), "remote-1") {
		t.Fatalf("output missing remote sandbox id:\n%s", out)
	}
}
