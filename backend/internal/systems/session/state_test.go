package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateTransition_Valid(t *testing.T) {
	err := ValidateTransition(StatePending, StatePlanning)
	assert.NoError(t, err)

	err = ValidateTransition(StatePlanning, StateProcessing)
	assert.NoError(t, err)

	err = ValidateTransition(StateProcessing, StateBlocked)
	assert.NoError(t, err)

	err = ValidateTransition(StateBlocked, StateProcessing)
	assert.NoError(t, err)

	err = ValidateTransition(StateProcessing, StateCompleted)
	assert.NoError(t, err)
}

func TestStateTransition_Invalid(t *testing.T) {
	err := ValidateTransition(StateCompleted, StatePending)
	assert.Error(t, err)

	err = ValidateTransition(StateProcessing, StatePending)
	assert.Error(t, err)
}

func TestCanTransition(t *testing.T) {
	assert.True(t, CanTransition(StatePending, StatePlanning))
	assert.True(t, CanTransition(StatePending, StateProcessing))
	assert.True(t, CanTransition(StatePlanning, StateBlocked))
	assert.False(t, CanTransition(StateCompleted, StatePending))
}
