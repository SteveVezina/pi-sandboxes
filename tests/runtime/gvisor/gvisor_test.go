// Package gvisor_test exercises the gVisor driver's exported surface
// (F18). It runs on every OS: on Linux it hits the real oci.CLIEngine
// wiring for lifecycle-when-unavailable paths (no runsc/docker required to
// assert an actionable error); on non-Linux it hits runtime_stub.go, which
// mirrors the same Driver contract and always reports unavailable.
package gvisor_test

import (
	"context"
	"testing"
	"time"

	pruntime "github.com/pi-sandbox/pi/pkg/runtime"
	"github.com/pi-sandbox/pi/pkg/runtime/gvisor"
)

func TestRuntimeName(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.Name() != "gvisor" {
		t.Errorf("Expected name 'gvisor', got '%s'", rt.Name())
	}
}

func TestRuntimeMode(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.Mode() != pruntime.ModeSecure {
		t.Errorf("Expected mode %q, got %q", pruntime.ModeSecure, rt.Mode())
	}
}

func TestRuntimeIsAvailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	// Just verify it doesn't panic; true/false depends on the host having
	// runsc + docker on PATH.
	_ = rt.IsAvailable()
}

func TestRuntimeNewWithDefaults(t *testing.T) {
	rt := gvisor.New("", "/tmp/gvisor-test", 0)
	if rt.Name() != gvisor.RuntimeName {
		t.Errorf("Expected name %q, got %q", gvisor.RuntimeName, rt.Name())
	}
}

func TestRuntimeProbe(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	report := rt.Probe(context.Background())
	if report.Available != rt.IsAvailable() {
		t.Errorf("Probe().Available = %v, want %v (IsAvailable)", report.Available, rt.IsAvailable())
	}
	if !report.Available && report.Reason == "" {
		t.Error("expected an actionable Reason when unavailable")
	}
}

func TestRuntimeCreate_Unavailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.IsAvailable() {
		t.Skip("gVisor is available on this host; unavailable-path test doesn't apply")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := rt.Create(ctx, pruntime.SandboxSpec{SandboxID: "test-create"})
	if err == nil {
		t.Fatal("Expected error when gVisor is not available")
	}
}

func TestRuntimeExec_Unavailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.IsAvailable() {
		t.Skip("gVisor is available on this host; unavailable-path test doesn't apply")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := rt.Exec(ctx, pruntime.Handle{SandboxID: "test-exec"}, pruntime.ExecRequest{Command: []string{"echo", "hello"}})
	if err == nil {
		t.Error("Expected error when gVisor is not available")
	}
}

func TestRuntimeDestroy_Unavailable(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.IsAvailable() {
		t.Skip("gVisor is available on this host; unavailable-path test doesn't apply")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Should fail gracefully, not panic.
	_ = rt.Destroy(ctx, pruntime.Handle{SandboxID: "test-destroy"})
}

func TestRuntimeStats_NotImplemented(t *testing.T) {
	rt := gvisor.Default("/tmp/gvisor-test")
	if rt.IsAvailable() {
		t.Skip("gVisor is available on this host; Stats() would attempt a real call")
	}
	_, err := rt.Stats(context.Background(), pruntime.Handle{SandboxID: "test-stats"})
	if err == nil {
		t.Error("Expected error: Stats is not implemented / gVisor unavailable")
	}
}
