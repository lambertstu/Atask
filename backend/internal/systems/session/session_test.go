package session

import (
	"path/filepath"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestSessionManager_CreateAndLoad(t *testing.T) {
	tempDir := t.TempDir()

	sm := NewSessionManager(tempDir)
	session := sm.CreateSession(tempDir, "glm-5")

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, tempDir, session.ProjectPath)
	assert.Equal(t, "glm-5", session.Model)
	assert.Equal(t, StatePending, session.State)
	assert.NotZero(t, session.CreatedAt)

	// Verify it was saved to disk
	savedFile := filepath.Join(tempDir, ".sessions", session.ID+".json")
	assert.FileExists(t, savedFile)

	// Verify retrieval
	retrieved := sm.GetSession(session.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.ID, retrieved.ID)
}

func TestSessionManager_ListSessions(t *testing.T) {
	tempDir := t.TempDir()

	sm := NewSessionManager(tempDir)
	sm.CreateSession(tempDir, "glm-5")
	sm.CreateSession(tempDir, "glm-5")

	// Should only list sessions for the specified project path
	list := sm.ListSessions(tempDir)
	assert.Len(t, list, 2)
}

func TestSessionManager_SubmitInput(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir)

	session := sm.CreateSession(tempDir, "glm-5")

	// Test valid submission to "plan" mode (default)
	err := sm.SubmitInput(session.ID, "hello world", "plan")
	assert.NoError(t, err)

	updated := sm.GetSession(session.ID)
	assert.Equal(t, "hello world", updated.Input)
	assert.Equal(t, StatePlanning, updated.State)
	assert.Len(t, updated.Messages, 1)
	assert.Equal(t, openai.ChatMessageRoleUser, updated.Messages[0].Role)
	assert.Equal(t, "hello world", updated.Messages[0].Content)

	// Test submission to "build" mode from planning state
	err = sm.SubmitInput(session.ID, "second input", "build")
	assert.NoError(t, err) // now allowed to transition to processing from planning

	updated2 := sm.GetSession(session.ID)
	assert.Equal(t, StateProcessing, updated2.State)
	assert.Len(t, updated2.Messages, 2)
}

func TestSessionManager_Transition(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir)

	session := sm.CreateSession(tempDir, "glm-4")
	assert.Equal(t, StatePending, session.State)

	// Pending -> Planning
	err := sm.Transition(session.ID, StatePlanning)
	assert.NoError(t, err)
	assert.Equal(t, StatePlanning, sm.GetSession(session.ID).State)

	// Planning -> Processing
	err = sm.Transition(session.ID, StateProcessing)
	assert.NoError(t, err)
	assert.Equal(t, StateProcessing, sm.GetSession(session.ID).State)

	// Processing -> Pending (Invalid)
	err = sm.Transition(session.ID, StatePending)
	assert.Error(t, err)
	assert.Equal(t, StateProcessing, sm.GetSession(session.ID).State) // state unchanged
}

func TestSessionManager_BlockUnblock(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir)

	session := sm.CreateSession(tempDir, "glm-4")
	// State must be Planning or Processing to transition to Blocked
	sm.Transition(session.ID, StatePlanning)

	// Test Block
	blockedArgs := map[string]interface{}{"cmd": "ls"}
	err := sm.SetBlocked(session.ID, "waiting for approval", "shell", blockedArgs)
	assert.NoError(t, err)

	updated := sm.GetSession(session.ID)
	assert.Equal(t, StateBlocked, updated.State)
	assert.Equal(t, "waiting for approval", updated.BlockedOn)
	assert.Equal(t, "shell", updated.BlockedTool)
	assert.Equal(t, blockedArgs, updated.BlockedArgs)

	// Test Unblock
	err = sm.Unblock(session.ID, "user approved")
	assert.NoError(t, err)

	unblocked := sm.GetSession(session.ID)
	assert.Equal(t, StateProcessing, unblocked.State)
	assert.Empty(t, unblocked.BlockedOn)
	assert.Empty(t, unblocked.BlockedTool)
	assert.Nil(t, unblocked.BlockedArgs)

	// Verify unblock response was added as user message
	assert.Len(t, unblocked.Messages, 1)
	assert.Equal(t, openai.ChatMessageRoleUser, unblocked.Messages[0].Role)
	assert.Equal(t, "user approved", unblocked.Messages[0].Content)
}

func TestSessionManager_UpdateMessages(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir)

	session := sm.CreateSession(tempDir, "glm-4")

	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "hi"},
		{Role: openai.ChatMessageRoleAssistant, Content: "hello"},
	}

	err := sm.UpdateMessages(session.ID, msgs)
	assert.NoError(t, err)

	updated := sm.GetSession(session.ID)
	assert.Len(t, updated.Messages, 2)
	assert.Equal(t, "hello", updated.Messages[1].Content)
}
