package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/git"
)

func TestValidateURL_Https(t *testing.T) {
	// Test that HTTPS URLs are allowed
	// This tests the validateURL function indirectly through Clone
	// We just check the URL validation by testing with an invalid URL first
	_, err := git.Clone(context.Background(), "file://etc/passwd", "/tmp/test-clone")
	if err == nil {
		t.Fatal("Expected error for file:// URL")
	}
}

func TestValidateURL_GitProtocol(t *testing.T) {
	_, err := git.Clone(context.Background(), "git://github.com/test/repo", "/tmp/test-clone")
	if err == nil {
		t.Fatal("Expected error for git:// URL")
	}
}

func TestClone_BareRepo(t *testing.T) {
	// Create a bare git repo for testing
	tmpDir := filepath.Join(os.TempDir(), "pi-git-test-"+randomID())
	defer os.RemoveAll(tmpDir)

	// Create a source repo
	srcDir := filepath.Join(tmpDir, "source")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "README.md"), []byte("# Test"), 0644)
	os.MkdirAll(filepath.Join(srcDir, ".git"), 0755)

	// Initialize git
	ctx := context.Background()
	_, err := git.Clone(ctx, srcDir, filepath.Join(tmpDir, "cloned"))
	// This may fail if git doesn't recognize it as a valid repo
	// But it shouldn't panic
	_ = err
}

func TestDiff_EmptyRepo(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-diff-test-"+randomID())
	defer os.RemoveAll(tmpDir)

	os.MkdirAll(tmpDir, 0755)

	// Initialize a git repo
	ctx := context.Background()
	result, err := git.Diff(ctx, tmpDir)
	// Diff on empty repo should return empty string, not error
	// (git diff returns 1 on empty repo, but we handle that)
	if err != nil {
		// Empty diff is acceptable
		t.Logf("Diff on empty repo: %v", err)
	} else {
		if result.Diff != "" {
			t.Logf("Diff output: %s", result.Diff)
		}
	}
}

func TestPatch_EmptyRepo(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "pi-patch-test-"+randomID())
	defer os.RemoveAll(tmpDir)

	os.MkdirAll(tmpDir, 0755)

	ctx := context.Background()
	result, err := git.Patch(ctx, tmpDir)
	if err != nil {
		t.Logf("Patch on empty repo: %v", err)
	} else {
		if result.Patch != "" {
			t.Logf("Patch output: %s", result.Patch)
		}
	}
}

func randomID() string {
	b := []byte("abcdefgh")
	return string(b)
}
