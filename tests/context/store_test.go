package context_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/context"
)

func TestStore_CreateAndLoadContext(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "contexts.yaml")

	store, err := context.NewStore(storePath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	ctx := context.Context{
		Name:      "workstation",
		Target:    "ssh://gpu-box.local",
		Transport: "ssh",
		Auth:      context.AuthConfig{Type: "ssh-agent"},
	}

	if err := store.Create(ctx); err != nil {
		t.Fatalf("Create: %v", err)
	}

	reloaded, err := context.NewStore(storePath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, err := reloaded.Get("workstation")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Target != ctx.Target {
		t.Fatalf("target = %q, want %q", got.Target, ctx.Target)
	}
	if got.Transport != "ssh" {
		t.Fatalf("transport = %q, want ssh", got.Transport)
	}
	if got.Auth.Type != "ssh-agent" {
		t.Fatalf("auth.type = %q, want ssh-agent", got.Auth.Type)
	}
}

func TestStore_LocalContextIsDefaultActive(t *testing.T) {
	dir := t.TempDir()
	store, err := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	active, err := store.Active()
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	if active.Name != "local" {
		t.Fatalf("active = %q, want local", active.Name)
	}
	if active.Transport != "unix" {
		t.Fatalf("local transport = %q, want unix", active.Transport)
	}
	if active.Auth.Type != "none" {
		t.Fatalf("local auth.type = %q, want none", active.Auth.Type)
	}
}

func TestStore_UseSwitchesActiveContext(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	if err := store.Create(context.Context{
		Name: "workstation", Target: "https://gpu-box.local:7777",
		Transport: "http", Auth: context.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TOKEN_WS"},
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Use("workstation"); err != nil {
		t.Fatalf("Use: %v", err)
	}
	active, _ := store.Active()
	if active.Name != "workstation" {
		t.Fatalf("active = %q, want workstation", active.Name)
	}
}

func TestStore_UseUnknownContextErrors(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	if err := store.Use("nope"); err == nil {
		t.Fatal("expected error for unknown context")
	}
}

func TestStore_DeleteContext(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	if err := store.Create(context.Context{Name: "tmp", Target: "https://x", Transport: "http", Auth: context.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TOKEN_TMP"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Delete("tmp"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get("tmp"); err == nil {
		t.Fatal("expected Get to fail after delete")
	}
}

func TestStore_DeleteActiveResetsToLocal(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	_ = store.Create(context.Context{Name: "ws", Target: "ssh://x", Transport: "ssh", Auth: context.AuthConfig{Type: "ssh-agent"}})
	_ = store.Use("ws")

	if err := store.Delete("ws"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	active, _ := store.Active()
	if active.Name != "local" {
		t.Fatalf("active after delete = %q, want local", active.Name)
	}
}

func TestStore_RejectsInvalidTransport(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))

	err := store.Create(context.Context{
		Name: "bad", Target: "x://y", Transport: "ftp", Auth: context.AuthConfig{Type: "none"},
	})
	if err == nil {
		t.Fatal("expected error for invalid transport")
	}
}

func TestStore_HTTPRequiresBearerToken(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))

	err := store.Create(context.Context{
		Name: "http-none", Target: "https://x", Transport: "http",
		Auth: context.AuthConfig{Type: "none"},
	})
	if err == nil {
		t.Fatal("expected http transport with auth=none to fail (ADR-003)")
	}
}

func TestStore_SSHRequiresSSHAgent(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))

	err := store.Create(context.Context{
		Name: "ssh-none", Target: "ssh://x", Transport: "ssh",
		Auth: context.AuthConfig{Type: "bearer-token"},
	})
	if err == nil {
		t.Fatal("expected ssh transport with non-ssh-agent auth to fail (ADR-003)")
	}
}

func TestStore_DoesNotStoreRawBearerToken(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "contexts.yaml")
	store, _ := context.NewStore(storePath)

	_ = store.Create(context.Context{
		Name: "ws", Target: "https://x", Transport: "http",
		Auth: context.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TOKEN_WS"},
	})

	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read contexts.yaml: %v", err)
	}
	if containsToken(string(data)) {
		t.Fatalf("contexts.yaml must not store raw tokens, got:\n%s", string(data))
	}
}

func containsToken(s string) bool {
	for _, fragment := range []string{"token:", "secret:", "password:"} {
		for i := 0; i+len(fragment) <= len(s); i++ {
			if s[i:i+len(fragment)] == fragment {
				return true
			}
		}
	}
	return false
}
