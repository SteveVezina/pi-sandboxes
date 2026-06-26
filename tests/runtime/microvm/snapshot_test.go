package microvm_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func TestWorkspaceDisk_IsExt4AndWritable(t *testing.T) {
	dir := t.TempDir()
	disk, err := microvm.CreateWorkspaceDisk(dir, "session-1", 64*1024*1024)
	if err != nil {
		t.Fatalf("CreateWorkspaceDisk: %v", err)
	}
	if disk.Filesystem != "ext4" {
		t.Fatalf("filesystem = %q, want ext4", disk.Filesystem)
	}
	if disk.ReadOnly {
		t.Fatal("workspace disk must be writable")
	}
	if !strings.HasPrefix(disk.Path, dir) {
		t.Fatalf("disk path %q not under temp dir %q", disk.Path, dir)
	}
}

func TestTemplateSnapshot_RestoreReturnsReadOnlyRootfs(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.snap")
	if err := microvm.WriteTemplateSnapshot(templatePath, []byte("dummy snapshot")); err != nil {
		t.Fatalf("WriteTemplateSnapshot: %v", err)
	}

	restore, err := microvm.RestoreTemplateSnapshot(templatePath)
	if err != nil {
		t.Fatalf("RestoreTemplateSnapshot: %v", err)
	}
	if !restore.Rootfs.ReadOnly {
		t.Fatal("restored rootfs must be read-only")
	}
	if restore.Rootfs.Path != templatePath {
		t.Fatalf("rootfs path = %q, want %q", restore.Rootfs.Path, templatePath)
	}
}

func TestReseedHook_RunsBeforeReady(t *testing.T) {
	var order []string
	hook := microvm.ReseedHook(func() error {
		order = append(order, "reseed")
		return nil
	})

	ready := func() error {
		order = append(order, "ready")
		return nil
	}

	if err := microvm.RunRestoreSequence(hook, ready); err != nil {
		t.Fatalf("RunRestoreSequence: %v", err)
	}

	if len(order) != 2 || order[0] != "reseed" || order[1] != "ready" {
		t.Fatalf("order = %v, want [reseed ready]", order)
	}
}

func TestReseedHook_FailureAbortsReady(t *testing.T) {
	var order []string
	hook := microvm.ReseedHook(func() error {
		order = append(order, "reseed")
		return microvm.ErrReseedFailed
	})
	ready := func() error {
		order = append(order, "ready")
		return nil
	}

	if err := microvm.RunRestoreSequence(hook, ready); err == nil {
		t.Fatal("expected error when reseed fails")
	}
	for _, step := range order {
		if step == "ready" {
			t.Fatal("ready must not run when reseed fails")
		}
	}
}
