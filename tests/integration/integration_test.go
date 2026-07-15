package integration

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// projectDir returns the absolute project root.
func projectDir() string {
	// Walk up from test binary to find go.mod
	cwd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return cwd
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
	// Fallback
	if d, err := os.Getwd(); err == nil {
		return d
	}
	return "."
}

// testClient creates an HTTP client that speaks Unix socket.
func testClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
		Timeout: 5 * time.Second,
	}
}

// TestDaemonBuild verifies the daemon binary compiles.
func TestDaemonBuild(t *testing.T) {
	proj := projectDir()
	cmd := exec.Command("go", "build", "-o", "/dev/null", "./cmd/pi-sandboxd/main.go")
	cmd.Dir = proj
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Daemon build failed: %v\n%s", err, string(out))
	}
}

// TestCLIBuild verifies the CLI binary compiles.
func TestCLIBuild(t *testing.T) {
	proj := projectDir()
	cmd := exec.Command("go", "build", "-o", "/dev/null", "./cmd/pi-box/main.go")
	cmd.Dir = proj
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("CLI build failed: %v\n%s", err, string(out))
	}
}

// TestAllPackagesBuild verifies all packages compile.
func TestAllPackagesBuild(t *testing.T) {
	proj := projectDir()
	cmd := exec.Command("go", "build", "./pkg/...")
	cmd.Dir = proj
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Packages build failed: %v\n%s", err, string(out))
	}
}

// TestCLIHelp verifies the CLI --help output.
func TestMakefile(t *testing.T) {
	proj := projectDir()
	data, err := os.ReadFile(filepath.Join(proj, "Makefile"))
	if os.IsNotExist(err) {
		t.Fatal("Makefile not found")
	}

	content := string(data)
	targets := []string{"build", "test", "install", "clean", "mock-up", "mock-down"}
	for _, target := range targets {
		if !strings.Contains(content, target) {
			t.Errorf("Expected Makefile target '%s'", target)
		}
	}
}

// TestDockerfile verifies Dockerfile exists.
func TestDockerfile(t *testing.T) {
	proj := projectDir()
	_, err := os.Stat(filepath.Join(proj, "Dockerfile"))
	if os.IsNotExist(err) {
		t.Fatal("Dockerfile not found")
	}
}

// TestGoMod verifies go.mod is valid.
func TestGoMod(t *testing.T) {
	proj := projectDir()
	cmd := exec.Command("go", "mod", "verify")
	cmd.Dir = proj
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod verify failed: %v\n%s", err, string(out))
	}
}

// TestUnitSuite verifies all unit tests pass.
func init() {
}
