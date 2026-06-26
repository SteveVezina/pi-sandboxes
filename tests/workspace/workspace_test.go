package workspace_test

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/workspace"
)

func TestNewManager(t *testing.T) {
	m := workspace.NewManager("test-id", workspace.ModeCopy)
	if m.Dir() == "" {
		t.Error("Expected non-empty workspace dir")
	}
	if m.Mode() != workspace.ModeCopy {
		t.Errorf("Expected ModeCopy, got %s", m.Mode())
	}
}

func TestNewManager_DefaultMode(t *testing.T) {
	m := workspace.NewManager("test-id", "")
	if m.Mode() != workspace.ModeCopy {
		t.Errorf("Expected default ModeCopy, got %s", m.Mode())
	}
}

func TestEnsureDir(t *testing.T) {
	m := workspace.NewManager("ensure-test-"+randomID(), workspace.ModeCopy)
	if err := m.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if _, err := os.Stat(m.Dir()); os.IsNotExist(err) {
		t.Fatal("Expected workspace dir to exist")
	}
}

func TestValidatePath_Valid(t *testing.T) {
	m := workspace.NewManager("valid-path-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	absPath, err := m.ValidatePath("src/main.go")
	if err != nil {
		t.Fatalf("ValidatePath failed: %v", err)
	}
	if !filepath.IsAbs(absPath) {
		t.Error("Expected absolute path")
	}
}

func TestValidatePath_TraversalParent(t *testing.T) {
	m := workspace.NewManager("traversal-test-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	_, err := m.ValidatePath("../etc/passwd")
	if err == nil {
		t.Fatal("Expected error for path traversal")
	}
}

func TestValidatePath_TraversalSlash(t *testing.T) {
	m := workspace.NewManager("traversal-test2-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	_, err := m.ValidatePath("/etc/passwd")
	if err == nil {
		t.Fatal("Expected error for absolute path")
	}
}

func TestValidatePath_TraversalNested(t *testing.T) {
	m := workspace.NewManager("traversal-test3-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	_, err := m.ValidatePath("src/../../etc/passwd")
	if err == nil {
		t.Fatal("Expected error for nested path traversal")
	}
}

func TestReadFile(t *testing.T) {
	m := workspace.NewManager("read-test-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	// Write a test file
	testFile := filepath.Join(m.Dir(), "test.txt")
	if err := os.WriteFile(testFile, []byte("hello workspace"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	data, err := m.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != "hello workspace" {
		t.Errorf("Expected 'hello workspace', got '%s'", string(data))
	}
}

func TestReadFile_NotFound(t *testing.T) {
	m := workspace.NewManager("read-notfound-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	_, err := m.ReadFile("nonexistent.txt")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestWriteFile(t *testing.T) {
	m := workspace.NewManager("write-test-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	// Write a file
	err := m.WriteFile("src/main.go", []byte("package main"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(filepath.Join(m.Dir(), "src/main.go"))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if string(data) != "package main" {
		t.Errorf("Expected 'package main', got '%s'", string(data))
	}
}

func TestWriteFile_CreatesParentDirs(t *testing.T) {
	m := workspace.NewManager("write-mkdir-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	// Write a file in a nested path that doesn't exist
	err := m.WriteFile("a/b/c/deep.txt", []byte("deep content"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(m.Dir(), "a/b/c/deep.txt"))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if string(data) != "deep content" {
		t.Errorf("Expected 'deep content', got '%s'", string(data))
	}
}

func TestWriteFile_Traversal(t *testing.T) {
	m := workspace.NewManager("write-traversal-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	err := m.WriteFile("../etc/hack", []byte("evil"))
	if err == nil {
		t.Fatal("Expected error for path traversal in write")
	}
}

func TestPull(t *testing.T) {
	m := workspace.NewManager("pull-test-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	// Create a file in workspace
	testFile := filepath.Join(m.Dir(), "pull-src.txt")
	if err := os.WriteFile(testFile, []byte("pull me"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Create destination
	destDir := filepath.Join(os.TempDir(), "pi-pull-dest-"+randomID())
	os.MkdirAll(destDir, 0755)
	defer os.RemoveAll(destDir)

	err := m.Pull("pull-src.txt", destDir)
	if err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	// Verify file was copied
	destFile := filepath.Join(destDir, "pull-src.txt")
	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if string(data) != "pull me" {
		t.Errorf("Expected 'pull me', got '%s'", string(data))
	}
}

func TestPush(t *testing.T) {
	m := workspace.NewManager("push-test-"+randomID(), workspace.ModeCopy)
	m.EnsureDir()

	// Create source file on host
	srcDir := filepath.Join(os.TempDir(), "pi-push-src-"+randomID())
	defer os.RemoveAll(srcDir)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	srcFile := filepath.Join(srcDir, "push-src.txt")
	if err := os.WriteFile(srcFile, []byte("push me"), 0644); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Push to workspace
	err := m.Push(srcFile, "push-dest.txt")
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Verify file was copied
	data, err := os.ReadFile(filepath.Join(m.Dir(), "push-dest.txt"))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if string(data) != "push me" {
		t.Errorf("Expected 'push me', got '%s'", string(data))
	}
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
