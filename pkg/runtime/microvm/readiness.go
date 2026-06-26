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
	sessionID string
	state     SandboxState
}

// NewReadinessTracker creates a tracker for one sandbox session.
func NewReadinessTracker(sessionID string) *ReadinessTracker {
	return &ReadinessTracker{
		sessionID: sessionID,
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
	if frame.SessionID != t.sessionID {
		return fmt.Errorf("unexpected ready session %q", frame.SessionID)
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
func WriteReady(w io.Writer, sessionID string) error {
	frame, err := NewReadyFrame("ready-1", sessionID)
	if err != nil {
		return err
	}
	return EncodeFrame(w, frame)
}
