package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agent-base/pkg/events"
	"agent-base/pkg/security"

	"github.com/sashabaranov/go-openai"
)

type PermissionDecision struct {
	Approved   bool
	AddAllowed string
}

type Session struct {
	ID          string                         `json:"id"`
	ProjectPath string                         `json:"project_path"`
	Model       string                         `json:"model"`
	State       SessionState                   `json:"state"`
	CreatedAt   time.Time                      `json:"created_at"`
	Input       string                         `json:"input,omitempty"`
	Messages    []openai.ChatCompletionMessage `json:"messages"`
	BlockedOn   string                         `json:"blocked_on,omitempty"`
	BlockedTool string                         `json:"blocked_tool,omitempty"`
	BlockedArgs map[string]interface{}         `json:"blocked_args,omitempty"`
	Mode        string                         `json:"mode"`

	PermissionMgr   *security.PermissionManager `json:"-"`
	BlockedResponse chan PermissionDecision     `json:"-"`
	Ctx             context.Context             `json:"-"`
	CancelFunc      context.CancelFunc          `json:"-"`
}

type SessionManager struct {
	sessionsDir string
	sessions    map[string]*Session
	mu          sync.RWMutex
	eventBus    *events.EventBus
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func NewSessionManager(projectPath string, eventBus *events.EventBus) *SessionManager {
	sessionsDir := filepath.Join(projectPath, ".sessions")
	os.MkdirAll(sessionsDir, 0755)

	sm := &SessionManager{
		sessionsDir: sessionsDir,
		sessions:    make(map[string]*Session),
		eventBus:    eventBus,
	}
	sm.loadAll()
	return sm
}

func (sm *SessionManager) loadAll() {
	files, _ := filepath.Glob(filepath.Join(sm.sessionsDir, "*.json"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		session.Ctx = ctx
		session.CancelFunc = cancel

		pm := security.NewPermissionManager(security.PlanMode, session.ProjectPath)
		blockingChan := make(chan security.BlockingRequest, 10)
		pm.SetBlockingChannel(blockingChan)
		session.PermissionMgr = pm
		session.BlockedResponse = make(chan PermissionDecision, 1)

		sm.sessions[session.ID] = &session
	}
}

func (sm *SessionManager) CreateSession(projectPath, model string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	pm := security.NewPermissionManager(security.PlanMode, projectPath)
	blockingChan := make(chan security.BlockingRequest, 10)
	pm.SetBlockingChannel(blockingChan)

	session := &Session{
		ID:              generateID(),
		ProjectPath:     projectPath,
		Model:           model,
		State:           StatePending,
		Mode:            security.PlanMode,
		CreatedAt:       time.Now(),
		Messages:        []openai.ChatCompletionMessage{},
		PermissionMgr:   pm,
		BlockedResponse: make(chan PermissionDecision, 1),
		Ctx:             ctx,
		CancelFunc:      cancel,
	}
	sm.sessions[session.ID] = session
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventStateChange, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"state":      session.State,
		}))
	}

	return session
}

func (sm *SessionManager) GetSession(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

func (sm *SessionManager) ListSessions(projectPath string) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var list []*Session
	for _, s := range sm.sessions {
		if filepath.Base(s.ProjectPath) == projectPath {
			list = append(list, s)
		}
	}
	return list
}

func (sm *SessionManager) SubmitInput(sessionID, input, mode string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if session.State == StatePlanning || session.State == StateProcessing {
		return fmt.Errorf("session is already running (state: %s), cannot submit new input", session.State)
	}

	session.Input = input
	session.Messages = append(session.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})

	newState := StatePlanning
	if mode == "build" {
		newState = StateProcessing
	}

	if err := ValidateTransition(session.State, newState); err != nil {
		return err
	}

	session.BlockedOn = ""
	session.BlockedTool = ""
	session.BlockedArgs = nil

	oldState := session.State
	session.State = newState
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventStateChange, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"old_state":  oldState,
			"new_state":  newState,
		}))
	}

	return nil
}

func (sm *SessionManager) Transition(sessionID string, newState SessionState, mode string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if err := ValidateTransition(session.State, newState); err != nil {
		return err
	}

	oldState := session.State
	session.Mode = mode
	session.State = newState
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventStateChange, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"old_state":  oldState,
			"new_state":  newState,
		}))
	}

	return nil
}

func (sm *SessionManager) SetBlocked(sessionID, blockedOn, blockedTool string, blockedArgs map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if err := ValidateTransition(session.State, StateBlocked); err != nil {
		return err
	}

	oldState := session.State
	session.State = StateBlocked
	session.BlockedOn = blockedOn
	session.BlockedTool = blockedTool
	session.BlockedArgs = blockedArgs
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventBlocked, session.ID, map[string]interface{}{
			"session_id":   session.ID,
			"old_state":    oldState,
			"blocked_on":   blockedOn,
			"blocked_tool": blockedTool,
			"blocked_args": blockedArgs,
		}))
	}

	return nil
}

func (sm *SessionManager) ClearBlockedState(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	var sessionState SessionState
	switch session.Mode {
	case security.PlanMode:
		sessionState = StatePlanning
	case security.BuildMode:
		sessionState = StateProcessing
	}

	if err := ValidateTransition(session.State, sessionState); err != nil {
		return err
	}

	oldState := session.State
	session.State = sessionState
	session.BlockedOn = ""
	session.BlockedTool = ""
	session.BlockedArgs = nil
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventStateChange, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"old_state":  oldState,
			"new_state":  StateProcessing,
			"unblocked":  true,
		}))
	}

	return nil
}

func (sm *SessionManager) UpdateMessages(sessionID string, messages []openai.ChatCompletionMessage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}
	session.Messages = messages
	sm.save(session)
	return nil
}

func (sm *SessionManager) save(session *Session) {
	path := filepath.Join(sm.sessionsDir, session.ID+".json")
	data, _ := json.MarshalIndent(session, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (sm *SessionManager) CompleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if err := ValidateTransition(session.State, StateCompleted); err != nil {
		return err
	}

	var sessionState SessionState
	switch session.Mode {
	case security.PlanMode:
		sessionState = StatePlanning
	case security.BuildMode:
		sessionState = StateCompleted
	}

	oldState := session.State
	session.State = sessionState
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventCompleted, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"old_state":  oldState,
		}))
	}

	return nil
}

func (sm *SessionManager) CancelSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventStateChange, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"state":      "cancelled",
		}))
	}

	return nil
}

func (sm *SessionManager) SubmitPermissionDecision(sessionID string, decision PermissionDecision) error {
	sm.mu.RLock()
	session := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("session not found")
	}

	if session.BlockedResponse == nil {
		return fmt.Errorf("session blocked response channel not initialized")
	}

	select {
	case session.BlockedResponse <- decision:
		return nil
	default:
		return fmt.Errorf("blocked response channel full or closed")
	}
}
