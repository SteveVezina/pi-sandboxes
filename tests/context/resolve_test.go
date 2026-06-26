package context_test

import (
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/context"
)

func TestResolve_UsesActiveContextByDefault(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	_ = store.Create(context.Context{
		Name: "ws", Target: "https://gpu-box.local:7777", Transport: "http",
		Auth: context.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TOKEN_WS"},
	})
	_ = store.Use("ws")

	got, err := store.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Name != "ws" {
		t.Fatalf("Resolve(\"\") = %q, want active context 'ws'", got.Name)
	}
}

func TestResolve_OverrideWinsOverActive(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))
	_ = store.Create(context.Context{
		Name: "ws", Target: "https://gpu-box.local:7777", Transport: "http",
		Auth: context.AuthConfig{Type: "bearer-token", TokenEnv: "PI_TOKEN_WS"},
	})
	_ = store.Use("ws")

	got, err := store.Resolve("local")
	if err != nil {
		t.Fatalf("Resolve(local): %v", err)
	}
	if got.Name != "local" {
		t.Fatalf("Resolve(local) = %q, want local", got.Name)
	}
}

func TestResolve_OverrideToUnknownErrors(t *testing.T) {
	dir := t.TempDir()
	store, _ := context.NewStore(filepath.Join(dir, "contexts.yaml"))

	_, err := store.Resolve("ghost")
	if err == nil {
		t.Fatal("expected error for unknown context override")
	}
}
