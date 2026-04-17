package session

import (
	"path/filepath"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestSessionManager_CreateAndLoad(t *testing.T) {
	tempDir := t.TempDir()

	sm := NewSessionManager(tempDir, nil)
	session := sm.CreateSession(tempDir, "glm-5")

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, tempDir, session.ProjectPath)
	assert.Equal(t, "glm-5", session.Model)
	assert.Equal(t, StatePending, session.State)
	assert.NotZero(t, session.CreatedAt)

	savedFile := filepath.Join(tempDir, ".sessions", session.ID+".json")
	assert.FileExists(t, savedFile)

	retrieved := sm.GetSession(session.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.ID, retrieved.ID)
}

func TestSessionManager_ListSessions(t *testing.T) {
	tempDir := t.TempDir()

	sm := NewSessionManager(tempDir, nil)
	sm.CreateSession(tempDir, "glm-5")
	sm.CreateSession(tempDir, "glm-5")

	list := sm.ListSessions(filepath.Base(tempDir))
	assert.Len(t, list, 2)
}

func TestSessionManager_SubmitInput(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

	session := sm.CreateSession(tempDir, "glm-5")

	err := sm.SubmitInput(session.ID, "hello world", "plan")
	assert.NoError(t, err)

	updated := sm.GetSession(session.ID)
	assert.Equal(t, "hello world", updated.Input)
	assert.Equal(t, StatePlanning, updated.State)
	assert.Len(t, updated.Messages, 1)
	assert.Equal(t, openai.ChatMessageRoleUser, updated.Messages[0].Role)
	assert.Equal(t, "hello world", updated.Messages[0].Content)

	err = sm.SubmitInput(session.ID, "second input", "build")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is already running")

	sm.CompleteSession(session.ID)

	err = sm.SubmitInput(session.ID, "restart input", "build")
	assert.NoError(t, err)

	updated2 := sm.GetSession(session.ID)
	assert.Equal(t, StateProcessing, updated2.State)
	assert.Len(t, updated2.Messages, 2)
	assert.Equal(t, "restart input", updated2.Messages[1].Content)
}

func TestSessionManager_Transition(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

	session := sm.CreateSession(tempDir, "glm-4")
	assert.Equal(t, StatePending, session.State)

	err := sm.Transition(session.ID, StatePlanning)
	assert.NoError(t, err)
	assert.Equal(t, StatePlanning, sm.GetSession(session.ID).State)

	err = sm.Transition(session.ID, StateProcessing)
	assert.NoError(t, err)
	assert.Equal(t, StateProcessing, sm.GetSession(session.ID).State)

	err = sm.Transition(session.ID, StatePending)
	assert.Error(t, err)
	assert.Equal(t, StateProcessing, sm.GetSession(session.ID).State)
}

func TestSessionManager_BlockUnblock(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

	session := sm.CreateSession(tempDir, "glm-4")
	sm.Transition(session.ID, StatePlanning)

	blockedArgs := map[string]interface{}{"cmd": "ls"}
	err := sm.SetBlocked(session.ID, "waiting for approval", "shell", blockedArgs)
	assert.NoError(t, err)

	updated := sm.GetSession(session.ID)
	assert.Equal(t, StateBlocked, updated.State)
	assert.Equal(t, "waiting for approval", updated.BlockedOn)
	assert.Equal(t, "shell", updated.BlockedTool)
	assert.Equal(t, blockedArgs, updated.BlockedArgs)

	decision := PermissionDecision{Approved: true}
	err = sm.SubmitPermissionDecision(session.ID, decision)
	assert.NoError(t, err)

	select {
	case res := <-session.BlockedResponse:
		assert.True(t, res.Approved)
	default:
		t.Fatal("expected decision in channel")
	}

	sm.Transition(session.ID, StateProcessing)
	unblocked := sm.GetSession(session.ID)
	assert.Equal(t, StateProcessing, unblocked.State)
}

func TestSessionManager_UpdateMessages(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

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

func TestSessionManager_Lifecycle(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

	session := sm.CreateSession(tempDir, "glm-5")
	assert.Equal(t, StatePending, session.State)

	err := sm.SubmitInput(session.ID, "test input", "plan")
	assert.NoError(t, err)

	runningSession := sm.GetSession(session.ID)
	assert.Equal(t, StatePlanning, runningSession.State)
	assert.Len(t, runningSession.Messages, 1)

	err = sm.SubmitInput(session.ID, "another input", "plan")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session is already running")

	err = sm.CompleteSession(session.ID)
	assert.NoError(t, err)

	completedSession := sm.GetSession(session.ID)
	assert.Equal(t, StateCompleted, completedSession.State)

	err = sm.SubmitInput(session.ID, "restart input", "build")
	assert.NoError(t, err)

	restartedSession := sm.GetSession(session.ID)
	assert.Equal(t, StateProcessing, restartedSession.State)
	assert.Len(t, restartedSession.Messages, 2)
	assert.Equal(t, "restart input", restartedSession.Messages[1].Content)
}

func TestSessionManager_PermissionMgrBlockingChannel(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

	session := sm.CreateSession(tempDir, "glm-5")

	assert.NotNil(t, session.PermissionMgr)
	assert.True(t, session.PermissionMgr.IsBlockingMode())
	assert.NotNil(t, session.PermissionMgr.GetBlockingChannel())

	assert.NotNil(t, session.BlockedResponse)
}

func TestSessionManager_LoadAllWithBlockingChannel(t *testing.T) {
	tempDir := t.TempDir()
	sm := NewSessionManager(tempDir, nil)

	session1 := sm.CreateSession(tempDir, "glm-5")
	session2 := sm.CreateSession(tempDir, "glm-4")

	assert.True(t, session1.PermissionMgr.IsBlockingMode())
	assert.True(t, session2.PermissionMgr.IsBlockingMode())

	sm2 := NewSessionManager(tempDir, nil)

	loaded1 := sm2.GetSession(session1.ID)
	loaded2 := sm2.GetSession(session2.ID)

	assert.NotNil(t, loaded1)
	assert.NotNil(t, loaded2)
	assert.True(t, loaded1.PermissionMgr.IsBlockingMode())
	assert.True(t, loaded2.PermissionMgr.IsBlockingMode())
}
