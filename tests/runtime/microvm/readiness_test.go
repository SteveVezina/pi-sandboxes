package microvm_test

import (
	"bufio"
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pi-sandbox/pi/pkg/runtime/microvm"
)

func TestReadinessTracker_ObserveReadyEvent_MarksWarm(t *testing.T) {
	tracker := microvm.NewReadinessTracker("sandbox-1")
	if tracker.Warm() {
		t.Fatal("new readiness tracker must not be warm")
	}

	hello, err := microvm.NewHelloFrame("hello-1", "sandbox-1")
	if err != nil {
		t.Fatalf("NewHelloFrame failed: %v", err)
	}
	if err := tracker.Observe(hello); err != nil {
		t.Fatalf("Observe hello failed: %v", err)
	}
	if tracker.Warm() {
		t.Fatal("hello frame must not mark sandbox warm")
	}

	ready, err := microvm.NewReadyFrame("ready-1", "sandbox-1")
	if err != nil {
		t.Fatalf("NewReadyFrame failed: %v", err)
	}
	if err := tracker.Observe(ready); err != nil {
		t.Fatalf("Observe ready failed: %v", err)
	}
	if !tracker.Warm() {
		t.Fatal("ready frame must mark sandbox warm")
	}
	if tracker.State() != microvm.SandboxStateWarm {
		t.Fatalf("state = %q, want %q", tracker.State(), microvm.SandboxStateWarm)
	}
}

func TestReadinessTracker_ObserveWrongSession_DoesNotMarkWarm(t *testing.T) {
	tracker := microvm.NewReadinessTracker("sandbox-1")
	ready, err := microvm.NewReadyFrame("ready-1", "sandbox-2")
	if err != nil {
		t.Fatalf("NewReadyFrame failed: %v", err)
	}

	err = tracker.Observe(ready)
	if err == nil {
		t.Fatal("expected wrong sandbox error")
	}
	if !strings.Contains(err.Error(), "unexpected ready sandbox") {
		t.Fatalf("error = %v, want unexpected ready sandbox", err)
	}
	if tracker.Warm() {
		t.Fatal("wrong-sandbox ready frame must not mark sandbox warm")
	}
}

func TestGuestInit_StartsAgentAndReportsReady(t *testing.T) {
	root := repoRoot(t)
	tmp := t.TempDir()
	agentd := filepath.Join(tmp, "pi-agentd")

	build := exec.Command("go", "build", "-o", agentd, "./cmd/pi-agentd")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build pi-agentd failed: %v\n%s", err, out)
	}

	run := exec.Command("go", "run", "./cmd/pi-init", "--sandbox", "sandbox-1", "--agentd", agentd)
	run.Dir = root
	run.Stdin = strings.NewReader("")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	run.Stdout = &stdout
	run.Stderr = &stderr
	if err := run.Run(); err != nil {
		t.Fatalf("pi-init failed: %v\n%s", err, stderr.String())
	}

	frame, err := microvm.DecodeFrame(bufio.NewReader(&stdout))
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if frame.Type != microvm.FrameTypeEvent || frame.Method != "ready" || frame.SandboxID != "sandbox-1" {
		t.Fatalf("frame = %+v, want ready event for sandbox-1", frame)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
}
