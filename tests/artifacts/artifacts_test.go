package artifacts_test

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/artifacts"
)

func TestList_EmptyWorkspace(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-empty-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	m := artifacts.NewManager(workspaceDir)
	files, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}

func TestList_WithArtifacts(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-list-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	// Create artifact files
	os.MkdirAll(filepath.Join(workspaceDir, "artifacts"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "artifacts", "build.tar.gz"), []byte("artifact data"), 0644)
	os.MkdirAll(filepath.Join(workspaceDir, "workspace", "dist"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "workspace", "dist", "app.js"), []byte("js code"), 0644)

	m := artifacts.NewManager(workspaceDir)
	files, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d: %v", len(files), files)
	}
}

func TestList_ExcludesEmptyDirs(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-emptydir-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	// Create an empty artifact directory
	os.MkdirAll(filepath.Join(workspaceDir, "artifacts"), 0755)

	m := artifacts.NewManager(workspaceDir)
	files, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files (empty dir excluded), got %d", len(files))
	}
}

func TestList_OnlyKnownLocations(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-locations-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	// Create files in known locations
	os.MkdirAll(filepath.Join(workspaceDir, "artifacts"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "artifacts", "out.txt"), []byte("data"), 0644)
	os.MkdirAll(filepath.Join(workspaceDir, "workspace", "coverage"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "workspace", "coverage", "index.html"), []byte("cov"), 0644)

	// Create file in non-known location
	os.WriteFile(filepath.Join(workspaceDir, "workspace", "README.md"), []byte("readme"), 0644)

	m := artifacts.NewManager(workspaceDir)
	files, err := m.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files from known locations, got %d", len(files))
	}
}

func TestPull(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-pull-src-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	// Create artifact files
	os.MkdirAll(filepath.Join(workspaceDir, "artifacts"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "artifacts", "report.xml"), []byte("report"), 0644)
	os.MkdirAll(filepath.Join(workspaceDir, "workspace", "dist"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "workspace", "dist", "app.js"), []byte("js"), 0644)

	hostDest := filepath.Join(os.TempDir(), "pi-artifacts-pull-dest-"+randomID())
	defer os.RemoveAll(hostDest)

	m := artifacts.NewManager(workspaceDir)
	if err := m.Pull(hostDest); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	// Verify files were copied
	data, err := os.ReadFile(filepath.Join(hostDest, "artifacts", "report.xml"))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if string(data) != "report" {
		t.Errorf("Expected 'report', got '%s'", string(data))
	}
}

func TestPull_EmptyWorkspace(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-pull-empty-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	hostDest := filepath.Join(os.TempDir(), "pi-artifacts-pull-empty-dest-"+randomID())
	defer os.RemoveAll(hostDest)

	m := artifacts.NewManager(workspaceDir)
	if err := m.Pull(hostDest); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
}

func TestPack(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-pack-src-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	// Create artifact files
	os.MkdirAll(filepath.Join(workspaceDir, "artifacts"), 0755)
	os.WriteFile(filepath.Join(workspaceDir, "artifacts", "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(workspaceDir, "artifacts", "file2.txt"), []byte("content2"), 0644)

	outputPath := filepath.Join(os.TempDir(), "pi-artifacts-pack-"+randomID()+".tar")
	defer os.Remove(outputPath)

	m := artifacts.NewManager(workspaceDir)
	if err := m.Pack(outputPath); err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Verify archive exists and has content
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Archive not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Expected non-empty archive")
	}

	// Verify tar contents
	f, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Open archive: %v", err)
	}
	defer f.Close()

	tr := tar.NewReader(f)
	fileCount := 0
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		fileCount++
		if h.Name == "file1.txt" {
			data := make([]byte, 8)
			tr.Read(data)
			if string(data) != "content1" {
				t.Errorf("Expected 'content1', got '%s'", string(data))
			}
		}
	}

	if fileCount < 2 {
		t.Errorf("Expected >= 2 files in archive, got %d", fileCount)
	}
}

func TestPack_EmptyWorkspace(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-pack-empty-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	outputPath := filepath.Join(os.TempDir(), "pi-artifacts-pack-empty-"+randomID()+".tar")
	defer os.Remove(outputPath)

	m := artifacts.NewManager(workspaceDir)
	if err := m.Pack(outputPath); err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// Should create an empty file
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Archive not found: %v", err)
	}
	if info.Size() != 0 {
		t.Error("Expected empty archive for empty workspace")
	}
}

func TestNewManager(t *testing.T) {
	workspaceDir := filepath.Join(os.TempDir(), "pi-artifacts-manager-"+randomID())
	os.MkdirAll(workspaceDir, 0755)
	defer os.RemoveAll(workspaceDir)

	m := artifacts.NewManager(workspaceDir)
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
}

func randomID() string {
	b := []byte("abcdefghijklmnopqrstuvwxyz012345")
	n := len(b)
	result := make([]byte, 8)
	for i := range result {
		result[i] = b[i%n]
	}
	return string(result)
}
