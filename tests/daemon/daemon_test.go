package daemon_test

import (
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/logs"
	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestDaemon_StartAndHealth(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sandboxd.sock")
	store := sandbox.NewStore(tmpDir)

	d := daemon.New(socketPath, 0, store)

	if err := d.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer d.Stop()

	time.Sleep(100 * time.Millisecond)

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Fatal("Socket file not created")
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	resp, err := client.Get("http://localhost/health")
	if err != nil {
		t.Fatalf("Health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"ok"}` {
		t.Errorf("Expected {\"status\":\"ok\"}, got %s", string(body))
	}
}

func TestDaemon_CreatesDir(t *testing.T) {
	// Use a short temp dir to stay under macOS 104-char socket path limit
	tmpDir := filepath.Join(os.TempDir(), "pi-test")
	os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir)
	// Use a nested path that doesn't exist yet
	socketPath := filepath.Join(tmpDir, "nonexistent", "sandboxd.sock")
	store := sandbox.NewStore(tmpDir)

	// Verify parent doesn't exist yet
	parentDir := filepath.Dir(socketPath)
	if _, err := os.Stat(parentDir); err == nil {
		t.Fatal("Parent directory should not exist yet")
	}

	d := daemon.New(socketPath, 0, store)

	if err := d.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer d.Stop()

	// Verify parent directory was created
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Fatal("Parent directory not created")
	}
}

func TestDaemon_HTTPPort(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sandboxd.sock")
	store := sandbox.NewStore(tmpDir)

	d := daemon.New(socketPath, 9999, store)

	if err := d.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer d.Stop()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:9999/health")
	if err != nil {
		t.Fatalf("HTTP health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestDaemon_EgressProxy_EnforcesSandboxPolicy(t *testing.T) {
	// Short base dir — macOS caps unix socket paths at ~104 chars and this
	// test's name makes t.TempDir() too long.
	tmpDir, err := os.MkdirTemp("", "pid")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	// logs.NewManager resolves via $HOME/.pi-box/sandboxes/<id>/logs; point
	// HOME here and put the store in the same tree so the egress log the
	// daemon writes is the one this test reads.
	t.Setenv("HOME", tmpDir)
	store := sandbox.NewStore(filepath.Join(tmpDir, ".pi-box", "sandboxes"))

	// One restricted sandbox (default allowlist) and one none-mode sandbox.
	restrictedID, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name: "r", Template: "base", Mode: "fast", NetworkMode: "restricted",
	})
	if err != nil {
		t.Fatal(err)
	}
	noneID, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name: "n", Template: "base", Mode: "fast", NetworkMode: "none",
	})
	if err != nil {
		t.Fatal(err)
	}

	d := daemon.New(filepath.Join(tmpDir, "sandboxd.sock"), 0, store)
	d.SetEgressProxyPort(free(t))
	if err := d.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer d.Stop()
	time.Sleep(100 * time.Millisecond)

	if d.ProxyAddr() == "" {
		t.Fatal("ProxyAddr empty after enabling egress proxy")
	}

	// github.com is on the default restricted allowlist → CONNECT accepted
	// (dial may still fail offline, but not with 403).
	if code := connectStatus(t, d.ProxyAddr(), restrictedID, "github.com:443"); code == 403 {
		t.Errorf("restricted sandbox: github.com should not be 403, got %d", code)
	}
	// evil.example.com is not allowlisted → 403.
	if code := connectStatus(t, d.ProxyAddr(), restrictedID, "evil.example.com:443"); code != 403 {
		t.Errorf("restricted sandbox: evil.example.com want 403, got %d", code)
	}
	// none mode → everything 403.
	if code := connectStatus(t, d.ProxyAddr(), noneID, "github.com:443"); code != 403 {
		t.Errorf("none sandbox: want 403, got %d", code)
	}

	// T30.6: the denial was recorded in the sandbox's egress log.
	events, err := logs.NewManager(restrictedID).EgressEvents()
	if err != nil {
		t.Fatalf("EgressEvents: %v", err)
	}
	var sawDenial bool
	for _, e := range events {
		if e.Host == "evil.example.com" && !e.Allowed {
			sawDenial = true
		}
		if e.Allowed {
			t.Errorf("egress log should hold only denials, got allow for %s", e.Host)
		}
	}
	if !sawDenial {
		t.Errorf("expected a recorded denial for evil.example.com, got %+v", events)
	}
}

func free(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func connectStatus(t *testing.T, proxyAddr, sandboxID, target string) int {
	t.Helper()
	c, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer c.Close()
	auth := base64.StdEncoding.EncodeToString([]byte(sandboxID + ":x"))
	_, _ = c.Write([]byte("CONNECT " + target + " HTTP/1.1\r\nHost: " + target +
		"\r\nProxy-Authorization: Basic " + auth + "\r\n\r\n"))
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 64)
	n, _ := c.Read(buf)
	line := string(buf[:n])
	switch {
	case containsSub(line, " 200 "):
		return 200
	case containsSub(line, " 403 "):
		return 403
	case containsSub(line, " 502 "), containsSub(line, " 400 "):
		return 502
	default:
		return 0
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestRouter_GUICORSPreflight(t *testing.T) {
	store := sandbox.NewStore(t.TempDir())
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodOptions, "/v1/sandboxes", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5174")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:5174" {
		t.Fatalf("expected GUI origin to be allowed, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected allowed methods header")
	}
}

func TestRouter_GUICORSHealth(t *testing.T) {
	store := sandbox.NewStore(t.TempDir())
	router := daemon.NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5174")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5174" {
		t.Fatalf("expected localhost GUI origin to be allowed, got %q", got)
	}
}

func TestDaemon_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sandboxd.sock")
	store := sandbox.NewStore(tmpDir)

	d := daemon.New(socketPath, 0, store)

	if err := d.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
