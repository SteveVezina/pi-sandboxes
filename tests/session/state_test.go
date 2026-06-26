package session_test

import (
	"testing"

	"github.com/pi-sandbox/pi/pkg/session"
)

func TestCanTransition_Valid(t *testing.T) {
	tests := []struct {
		from, to session.State
	}{
		{session.StateCreating, session.StateWarm},
		{session.StateWarm, session.StateExecuting},
		{session.StateWarm, session.StateDestroying},
		{session.StateExecuting, session.StateWarm},
		{session.StateExecuting, session.StateDestroying},
		{session.StateDestroying, session.StateDestroyed},
	}

	for _, tt := range tests {
		if !session.CanTransition(tt.from, tt.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", tt.from, tt.to)
		}
	}
}

func TestCanTransition_Invalid(t *testing.T) {
	tests := []struct {
		from, to session.State
	}{
		{session.StateWarm, session.StateCreating},
		{session.StateCreating, session.StateExecuting},
		{session.StateDestroying, session.StateWarm},
		{session.StateDestroyed, session.StateWarm},
		{session.StateExecuting, session.StateCreating},
	}

	for _, tt := range tests {
		if session.CanTransition(tt.from, tt.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", tt.from, tt.to)
		}
	}
}

func TestValidateTransition_Valid(t *testing.T) {
	err := session.ValidateTransition(session.StateWarm, session.StateExecuting)
	if err != nil {
		t.Fatalf("Expected nil error, got: %v", err)
	}
}

func TestValidateTransition_Invalid(t *testing.T) {
	err := session.ValidateTransition(session.StateDestroyed, session.StateWarm)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	_, ok := err.(*session.TransitionError)
	if !ok {
		t.Errorf("Expected TransitionError, got %T", err)
	}
}
