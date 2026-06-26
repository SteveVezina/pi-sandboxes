package daemon_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pi-sandbox/pi/pkg/daemon"
	"github.com/pi-sandbox/pi/pkg/session"
)

func TestDaemon_StartAndHealth(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sandboxd.sock")
	store := session.NewStore(tmpDir)

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
	store := session.NewStore(tmpDir)

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
	store := session.NewStore(tmpDir)

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

func TestDaemon_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "sandboxd.sock")
	store := session.NewStore(tmpDir)

	d := daemon.New(socketPath, 0, store)

	if err := d.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
