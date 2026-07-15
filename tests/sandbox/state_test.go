package sandbox_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/sandbox"
)

func TestCanTransition_Valid(t *testing.T) {
	tests := []struct {
		from, to sandbox.State
	}{
		{sandbox.StateCreating, sandbox.StateWarm},
		{sandbox.StateWarm, sandbox.StateExecuting},
		{sandbox.StateWarm, sandbox.StateDestroying},
		{sandbox.StateExecuting, sandbox.StateWarm},
		{sandbox.StateExecuting, sandbox.StateDestroying},
		{sandbox.StateDestroying, sandbox.StateDestroyed},
	}

	for _, tt := range tests {
		if !sandbox.CanTransition(tt.from, tt.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", tt.from, tt.to)
		}
	}
}

func TestCanTransition_Invalid(t *testing.T) {
	tests := []struct {
		from, to sandbox.State
	}{
		{sandbox.StateWarm, sandbox.StateCreating},
		{sandbox.StateCreating, sandbox.StateExecuting},
		{sandbox.StateDestroying, sandbox.StateWarm},
		{sandbox.StateDestroyed, sandbox.StateWarm},
		{sandbox.StateExecuting, sandbox.StateCreating},
	}

	for _, tt := range tests {
		if sandbox.CanTransition(tt.from, tt.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", tt.from, tt.to)
		}
	}
}

func TestValidateTransition_Valid(t *testing.T) {
	err := sandbox.ValidateTransition(sandbox.StateWarm, sandbox.StateExecuting)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
}

func TestValidateTransition_Invalid(t *testing.T) {
	err := sandbox.ValidateTransition(sandbox.StateDestroyed, sandbox.StateWarm)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	_, ok := err.(*sandbox.TransitionError)
	if !ok {
		t.Errorf("Expected TransitionError, got %T", err)
	}
}
