package api

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestManagedVolumeName_SanitizesForOCIRuntime(t *testing.T) {
	got := managedVolumeName("workspace", "abc/def ghi", "npm@cache")
	want := "pi-sandbox-workspace-abc-def-ghi-npm-cache"
	if got != want {
		t.Fatalf("managedVolumeName() = %q, want %q", got, want)
	}
}

func TestCompatWorkspaceSource_UsesManagedVolumeForNonBindModes(t *testing.T) {
	id := "sandbox-123"
	for _, meta := range []*sandbox.Meta{
		{WorkspaceMode: "copy"},
		{WorkspaceMode: "copy", Workspace: "/workspace"},
		{WorkspaceMode: "overlay", Workspace: "/workspace"},
		nil,
	} {
		got := compatWorkspaceSource(id, meta)
		want := "pi-sandbox-workspace-sandbox-123"
		if got != want {
			t.Fatalf("compatWorkspaceSource(%+v) = %q, want %q", meta, got, want)
		}
	}
}

func TestCompatWorkspaceSource_UsesExplicitBindSource(t *testing.T) {
	meta := &sandbox.Meta{
		WorkspaceMode: "bind",
		Workspace:     "/tmp/project",
	}
	got := compatWorkspaceSource("sandbox-123", meta)
	if got != "/tmp/project" {
		t.Fatalf("compatWorkspaceSource() = %q, want explicit bind source", got)
	}
}
