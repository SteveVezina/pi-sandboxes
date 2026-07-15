package runtime_test

import (
	"context"
	"testing"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
)

type fakeProber struct {
	mode   pruntime.Mode
	report pruntime.CapabilityReport
	calls  int
}

func (f *fakeProber) Mode() pruntime.Mode { return f.mode }
func (f *fakeProber) Probe(ctx context.Context) pruntime.CapabilityReport {
	f.calls++
	return f.report
}

func TestRegistry_Register_RejectsDuplicateMode(t *testing.T) {
	reg := pruntime.NewRegistry()
	if err := reg.Register(&fakeProber{mode: pruntime.ModeFast}); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}
	if err := reg.Register(&fakeProber{mode: pruntime.ModeFast}); err == nil {
		t.Fatal("expected duplicate mode registration to fail")
	}
}

func TestRegistry_Reports_PreservesRegistrationOrder(t *testing.T) {
	reg := pruntime.NewRegistry()
	order := []pruntime.Mode{pruntime.ModeMicroVM, pruntime.ModeSecure, pruntime.ModeFast, pruntime.ModeCompat}
	for _, m := range order {
		if err := reg.Register(&fakeProber{
			mode:   m,
			report: pruntime.CapabilityReport{Mode: string(m)},
		}); err != nil {
			t.Fatalf("Register(%s) failed: %v", m, err)
		}
	}

	reports := reg.Reports(context.Background())
	if len(reports) != len(order) {
		t.Fatalf("expected %d reports, got %d", len(order), len(reports))
	}
	for i, m := range order {
		if reports[i].Mode != string(m) {
			t.Errorf("report[%d].Mode = %s, want %s", i, reports[i].Mode, m)
		}
	}
}

func TestRegistry_Reports_ExecutesProbeEveryCall(t *testing.T) {
	reg := pruntime.NewRegistry()
	fp := &fakeProber{mode: pruntime.ModeFast, report: pruntime.CapabilityReport{Mode: "fast", Available: true}}
	if err := reg.Register(fp); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	reg.Reports(context.Background())
	reg.Reports(context.Background())

	if fp.calls != 2 {
		t.Errorf("expected probe executed on every Reports call (2), got %d", fp.calls)
	}
}

func TestRegistry_Probe_UnknownModeReportsAbsent(t *testing.T) {
	reg := pruntime.NewRegistry()
	if _, ok := reg.Probe(context.Background(), pruntime.ModeMicroVM); ok {
		t.Fatal("expected Probe on unregistered mode to report absent")
	}
}
