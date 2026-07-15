package api

import "testing"

func TestManagedVolumeName_SanitizesForOCIRuntime(t *testing.T) {
	got := managedVolumeName("workspace", "abc/def ghi", "npm@cache")
	want := "pi-sandbox-workspace-abc-def-ghi-npm-cache"
	if got != want {
		t.Fatalf("managedVolumeName() = %q, want %q", got, want)
	}
}
