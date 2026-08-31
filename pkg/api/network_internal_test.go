package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func newNetTestStore(t *testing.T) *sandbox.Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "sandboxes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return sandbox.NewStore(dir)
}

func TestSandboxNetworkPolicy_PersistedModeRetrievableByID(t *testing.T) {
	store := newNetTestStore(t)
	id, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name: "n", Template: "base", Mode: "fast",
		NetworkMode: "restricted", NetworkAllow: []string{"internal.corp"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	p, err := sandboxNetworkPolicy(store, id)
	if err != nil {
		t.Fatalf("sandboxNetworkPolicy: %v", err)
	}
	if !p.IsAllowed("internal.corp") {
		t.Error("per-sandbox allowlist host should be retrievable and permitted")
	}
	if !p.IsAllowed("github.com") {
		t.Error("restricted default allowlist should still apply")
	}
	if p.IsAllowed("169.254.169.254") {
		t.Error("default deny must apply")
	}
}

func TestSandboxNetworkPolicy_NoneModeBlocksAll(t *testing.T) {
	store := newNetTestStore(t)
	id, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name: "n", Template: "base", Mode: "fast", NetworkMode: "none",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	p, err := sandboxNetworkPolicy(store, id)
	if err != nil {
		t.Fatalf("sandboxNetworkPolicy: %v", err)
	}
	if p.IsAllowed("github.com") {
		t.Error("none mode should block all egress")
	}
}

func TestSandboxNetworkPolicy_LegacySandboxDefaultsRestricted(t *testing.T) {
	store := newNetTestStore(t)
	// Simulate a pre-ADR-006 sandbox: no NetworkMode persisted.
	id, err := store.CreateWithOptions(sandbox.CreateOptions{
		Name: "legacy", Template: "base", Mode: "fast",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// NewMeta now defaults NetworkMode to "restricted"; force the empty case
	// by rewriting meta.json as a pre-ADR-006 sandbox would have it.
	meta, _ := store.Get(id)
	meta.NetworkMode = ""
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(store.Dir(), id, "meta.json"), data, 0o644); err != nil {
		t.Fatalf("rewrite meta: %v", err)
	}

	p, err := sandboxNetworkPolicy(store, id)
	if err != nil {
		t.Fatalf("sandboxNetworkPolicy: %v", err)
	}
	if p.IsAllowed("evil.example.com") {
		t.Error("legacy sandbox should fall back to restricted, not open")
	}
}
