package sandbox

import (
	"fmt"
)

// validTransitions defines allowed state transitions.
var validTransitions = map[State][]State{
	StateCreating:   {StateWarm},
	StateWarm:       {StateExecuting, StateDestroying},
	StateExecuting:  {StateWarm, StateDestroying},
	StateDestroying: {StateDestroyed},
	// StateDestroyed has no outgoing transitions (terminal state)
}

// TransitionError is returned when an invalid state transition is attempted.
type TransitionError struct {
	From State
	To   State
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("invalid state transition: %s → %s", e.From, e.To)
}

// CanTransition checks if a transition from `from` to `to` is valid.
func CanTransition(from, to State) bool {
	valid, ok := validTransitions[from]
	if !ok {
		return false // unknown source state
	}
	for _, v := range valid {
		if v == to {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if the transition is invalid.
func ValidateTransition(from, to State) error {
	if !CanTransition(from, to) {
		return &TransitionError{From: from, To: to}
	}
	return nil
}
