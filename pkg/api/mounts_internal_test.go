package api

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

// AC-34.1: no sandbox has a writable bind mount of any host directory by default.
// The compat create/exec paths must derive workspace, artifacts, and cache
// sources from daemon-managed named volumes, never from host paths, unless the
// caller explicitly opts in to bind mode.

func isHostPath(source string) bool {
	return filepath.IsAbs(source)
}

func TestManagedVolumeName_AnyInput_NeverHostPath(t *testing.T) {
	cases := []struct {
		name  string
		parts []string
	}{
		{"workspace", []string{"workspace", "sbx-123"}},
		{"artifacts", []string{"artifacts", "sbx-123"}},
		{"cache with slashes", []string{"cache", "sbx-123", "/etc/passwd"}},
		{"cache with traversal", []string{"cache", "..", "..", "home"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := managedVolumeName(tc.parts...)
			if isHostPath(got) {
				t.Fatalf("managedVolumeName(%v) = %q, want non-absolute volume name", tc.parts, got)
			}
			if got == "" {
				t.Fatalf("managedVolumeName(%v) = empty", tc.parts)
			}
		})
	}
}

func TestCompatWorkspaceSource_DefaultMode_UsesManagedVolume(t *testing.T) {
	// nil meta and non-bind modes must never surface a host directory.
	for _, meta := range []*sandbox.Meta{
		nil,
		{WorkspaceMode: "copy", Workspace: "/home/user/project"},
		{WorkspaceMode: "", Workspace: "/home/user/project"},
	} {
		got := compatWorkspaceSource("sbx-123", meta)
		if isHostPath(got) {
			t.Fatalf("compatWorkspaceSource(meta=%+v) = %q, want managed volume, not host path", meta, got)
		}
	}
}

func TestCompatWorkspaceSource_BindMode_OptsInToHostPath(t *testing.T) {
	meta := &sandbox.Meta{WorkspaceMode: "bind", Workspace: "/home/user/project"}
	got := compatWorkspaceSource("sbx-123", meta)
	if got != "/home/user/project" {
		t.Fatalf("compatWorkspaceSource(bind) = %q, want explicit host path opt-in", got)
	}
}

func TestCompatArtifactsSource_Always_UsesManagedVolume(t *testing.T) {
	got := compatArtifactsSource("sbx-123")
	if isHostPath(got) {
		t.Fatalf("compatArtifactsSource = %q, want managed volume, not host path", got)
	}
}

func TestCreateRequest_HasNoHostMountFields(t *testing.T) {
	// Guard against regressions: CreateRequest must not grow a field that lets a
	// caller inject an arbitrary host bind mount without going through the
	// explicit workspace bind opt-in.
	allowed := map[string]bool{
		"Template": true, "Mode": true, "Name": true, "TTL": true, "Workspace": true,
		"Network": true, // ADR-006: egress mode + allowlist hosts, no filesystem exposure
	}
	rt := reflect.TypeOf(CreateRequest{})
	for i := 0; i < rt.NumField(); i++ {
		if !allowed[rt.Field(i).Name] {
			t.Fatalf("CreateRequest gained unexpected field %q — audit it for host-mount exposure", rt.Field(i).Name)
		}
	}
}

// ADR-009: cache volumes are scoped by template/runtime/user, so sibling
// sandboxes of the same template share a warm cache — not per sandbox ID.
func TestCacheVolumeName_ScopedByTemplateNotSandbox(t *testing.T) {
	a := cacheVolumeName("node-python", "compat", "npm")
	b := cacheVolumeName("node-python", "compat", "npm")
	if a != b {
		t.Fatalf("same template/runtime/type must yield the same volume: %q vs %q", a, b)
	}
	if isHostPath(a) {
		t.Fatalf("cache volume name must not be a host path: %q", a)
	}

	if cacheVolumeName("go", "compat", "GOMODCACHE") == a {
		t.Error("different template should give a different cache volume")
	}
	if cacheVolumeName("node-python", "fast", "npm") == a {
		t.Error("different runtime should give a different cache volume")
	}
	// No sandbox ID anywhere in the name.
	if strings.Contains(a, "sbx") || len(a) > 80 {
		t.Errorf("unexpected volume name shape: %q", a)
	}
}
