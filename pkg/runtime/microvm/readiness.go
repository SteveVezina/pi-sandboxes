package microvm

import (
	"fmt"
	"io"
)

// SandboxState is the host-observed lifecycle state for a microVM sandbox.
type SandboxState string

const (
	SandboxStateBooting SandboxState = "booting"
	SandboxStateWarm    SandboxState = "warm"
)

// ReadinessTracker maps guest lifecycle frames to host sandbox state.
type ReadinessTracker struct {
	sandboxID string
	state     SandboxState
}

// NewReadinessTracker creates a tracker for one sandbox.
func NewReadinessTracker(sandboxID string) *ReadinessTracker {
	return &ReadinessTracker{
		sandboxID: sandboxID,
		state:     SandboxStateBooting,
	}
}

// Observe updates state from a guest lifecycle frame.
func (t *ReadinessTracker) Observe(frame Frame) error {
	if frame.Method != "ready" {
		return nil
	}
	if frame.Type != FrameTypeEvent {
		return fmt.Errorf("ready must be an event frame")
	}
	if frame.SandboxID != t.sandboxID {
		return fmt.Errorf("unexpected ready sandbox %q", frame.SandboxID)
	}
	t.state = SandboxStateWarm
	return nil
}

// State returns the current host-observed sandbox state.
func (t *ReadinessTracker) State() SandboxState {
	return t.state
}

// Warm reports whether the guest has sent its ready event.
func (t *ReadinessTracker) Warm() bool {
	return t.state == SandboxStateWarm
}

// WriteReady writes the guest ready event to the control stream.
func WriteReady(w io.Writer, sandboxID string) error {
	frame, err := NewReadyFrame("ready-1", sandboxID)
	if err != nil {
		return err
	}
	return EncodeFrame(w, frame)
}
