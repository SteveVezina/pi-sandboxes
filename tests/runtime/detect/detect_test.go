package detect_test

import (
	"testing"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/detect"
)

func TestReports_CoversAllModesInPriorityOrder(t *testing.T) {
	reports := detect.Reports(t.TempDir())

	want := []string{"microvm", "secure", "fast", "compat"}
	if len(reports) != len(want) {
		t.Fatalf("expected %d reports, got %d", len(want), len(reports))
	}
	for i, mode := range want {
		if reports[i].Mode != mode {
			t.Errorf("report[%d].Mode = %s, want %s", i, reports[i].Mode, mode)
		}
	}
}

func TestReports_UnavailableModeHasReason(t *testing.T) {
	reports := detect.Reports(t.TempDir())

	for _, rep := range reports {
		if !rep.Available && rep.Reason == "" {
			t.Errorf("unavailable mode %s must carry a reason", rep.Mode)
		}
		if rep.Description == "" {
			t.Errorf("mode %s must carry a description", rep.Mode)
		}
	}
}

func TestReports_TiersAreSeparateAxes(t *testing.T) {
	reports := detect.Reports(t.TempDir())

	for _, rep := range reports {
		if rep.IsolationTier < 1 || rep.IsolationTier > 4 {
			t.Errorf("mode %s isolation tier out of range: %d", rep.Mode, rep.IsolationTier)
		}
		if rep.CompatTier < 1 {
			t.Errorf("mode %s compat tier missing: %d", rep.Mode, rep.CompatTier)
		}
	}
}

func TestAvailableRuntimes_MatchesReports(t *testing.T) {
	tmpDir := t.TempDir()
	available := detect.AvailableRuntimes(tmpDir)
	reports := detect.Reports(tmpDir)

	availFromReports := map[string]bool{}
	for _, rep := range reports {
		if rep.Available {
			availFromReports[rep.Mode] = true
		}
	}
	if len(available) != len(availFromReports) {
		t.Fatalf("AvailableRuntimes %v disagrees with reports %v", available, availFromReports)
	}
	for _, mode := range available {
		if !availFromReports[mode] {
			t.Errorf("mode %s listed available but report says otherwise", mode)
		}
	}
}

func TestBestMode_IsFirstAvailableInPriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()
	best := detect.BestMode(tmpDir)

	reports := detect.Reports(tmpDir)
	expected := "unknown"
	for _, rep := range reports {
		if rep.Available {
			expected = rep.Mode
			break
		}
	}
	if best != expected {
		t.Errorf("BestMode = %s, want first available %s", best, expected)
	}
}

func TestBestMode_Consistency(t *testing.T) {
	tmpDir := t.TempDir()
	if m1, m2 := detect.BestMode(tmpDir), detect.BestMode(tmpDir); m1 != m2 {
		t.Errorf("expected consistent mode, got %s vs %s", m1, m2)
	}
}

func TestDefaultRegistry_ProbesByMode(t *testing.T) {
	reg := detect.DefaultRegistry(t.TempDir())

	rep, ok := reg.Probe(t.Context(), pruntime.ModeCompat)
	if !ok {
		t.Fatal("expected compat prober registered")
	}
	if rep.Mode != "compat" {
		t.Errorf("expected compat report, got %s", rep.Mode)
	}
	if !rep.OCIImages {
		t.Error("compat mode must report OCI image support")
	}
}
