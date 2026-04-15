package session

import "fmt"

type SessionState string

const (
	StatePending    SessionState = "pending"
	StatePlanning   SessionState = "planning"
	StateProcessing SessionState = "processing"
	StateBlocked    SessionState = "blocked"
	StateCompleted  SessionState = "completed"
)

var ValidTransitions = map[SessionState][]SessionState{
	StatePending:    {StatePlanning, StateProcessing},
	StatePlanning:   {StateProcessing, StateBlocked, StateCompleted, StatePlanning},
	StateProcessing: {StateBlocked, StateCompleted, StatePlanning, StateProcessing},
	StateBlocked:    {StateProcessing, StateCompleted, StatePlanning},
	StateCompleted:  {StatePlanning, StateProcessing},
}

func CanTransition(from, to SessionState) bool {
	allowed := ValidTransitions[from]
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func ValidateTransition(from, to SessionState) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid state transition: %s -> %s", from, to)
	}
	return nil
}
