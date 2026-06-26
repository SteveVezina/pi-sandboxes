package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/detect"
)

func TestDetect(t *testing.T) {
	tmpDir := t.TempDir()
	rt, err := detect.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if rt == nil {
		t.Fatal("Expected non-nil runtime")
	}
	// Verify the runtime has a valid name
	name := rt.Name()
	if name == "" {
		t.Error("Expected non-empty runtime name")
	}
	// Verify it's available
	if !rt.IsAvailable() {
		t.Error("Expected runtime to be available")
	}
}

func TestDetect_ReturnsBestAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	rt, err := detect.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// The best runtime should have the highest security level
	mode := rt.GetMode()
	level := rt.GetSecurityLevel()

	// At minimum, we should get something
	if mode == "" {
		t.Error("Expected non-empty mode")
	}
	if level <= 0 {
		t.Errorf("Expected positive security level, got %d", level)
	}
}

func TestDetect_CleanTempDir(t *testing.T) {
	// Create a temp dir with files to ensure Detect handles it
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	rt, err := detect.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	_ = rt
	// Detect should not be affected by stray files
}

func TestAvailableRuntimes(t *testing.T) {
	tmpDir := t.TempDir()
	available := detect.AvailableRuntimes(tmpDir)

	if len(available) == 0 {
		t.Error("Expected at least one available runtime")
	}

	// Verify the best runtime mode is in the available list
	best, err := detect.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	// The best runtime mode should be in the available list
	bestMode := best.GetMode()
	found := false
	for _, name := range available {
		if name == bestMode {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Best runtime mode %s not in available list %v", bestMode, available)
	}
}

func TestBestMode(t *testing.T) {
	tmpDir := t.TempDir()
	mode := detect.BestMode(tmpDir)

	if mode == "" {
		t.Error("Expected non-empty mode")
	}
	if mode == "unknown" {
		t.Error("Expected a known mode, got 'unknown'")
	}
}

func TestBestSecurityLevel(t *testing.T) {
	tmpDir := t.TempDir()
	level := detect.BestSecurityLevel(tmpDir)

	if level <= 0 {
		t.Errorf("Expected positive security level, got %d", level)
	}
}

func TestDetect_NoRootDir(t *testing.T) {
	// Detect should work even with an empty root dir
	rt, err := detect.Detect("")
	if err != nil {
		t.Fatalf("Detect with empty root dir failed: %v", err)
	}
	if rt == nil {
		t.Fatal("Expected non-nil runtime")
	}
}

func TestDetect_MultipleRuns(t *testing.T) {
	tmpDir := t.TempDir()

	// Run Detect multiple times — should be consistent
	rt1, err := detect.Detect(tmpDir)
	if err != nil {
		t.Fatalf("First Detect failed: %v", err)
	}

	rt2, err := detect.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Second Detect failed: %v", err)
	}

	if rt1.Name() != rt2.Name() {
		t.Errorf("Expected consistent runtime name, got %s vs %s", rt1.Name(), rt2.Name())
	}
}

func TestDetect_AvailableRuntimesConsistency(t *testing.T) {
	tmpDir := t.TempDir()

	// Run AvailableRuntimes twice — should be consistent
	avail1 := detect.AvailableRuntimes(tmpDir)
	avail2 := detect.AvailableRuntimes(tmpDir)

	if len(avail1) != len(avail2) {
		t.Errorf("Expected consistent available count, got %d vs %d", len(avail1), len(avail2))
	}
}

func TestDetect_BestModeConsistency(t *testing.T) {
	tmpDir := t.TempDir()

	mode1 := detect.BestMode(tmpDir)
	mode2 := detect.BestMode(tmpDir)

	if mode1 != mode2 {
		t.Errorf("Expected consistent mode, got %s vs %s", mode1, mode2)
	}
}

func TestDetect_BestSecurityLevelConsistency(t *testing.T) {
	tmpDir := t.TempDir()

	level1 := detect.BestSecurityLevel(tmpDir)
	level2 := detect.BestSecurityLevel(tmpDir)

	if level1 != level2 {
		t.Errorf("Expected consistent security level, got %d vs %d", level1, level2)
	}
}
