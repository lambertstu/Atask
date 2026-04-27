package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type SessionRuntime struct {
	PermissionMgr   *security.PermissionManager
	BlockedResponse chan PermissionDecision
	Ctx             context.Context
	CancelFunc      context.CancelFunc
}

type SessionManager struct {
	sessionsDir string
	mu          sync.RWMutex
	eventBus    *events.EventBus

	runtimeMu  sync.Mutex
	runtimeMap map[string]*SessionRuntime
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func NewSessionManager(projectPath string, eventBus *events.EventBus) *SessionManager {
	sessionsDir := filepath.Join(projectPath, ".sessions")
	os.MkdirAll(sessionsDir, 0755)

	return &SessionManager{
		sessionsDir: sessionsDir,
		eventBus:    eventBus,
		runtimeMap:  make(map[string]*SessionRuntime),
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
	session, _ := sm.loadFromFile(id)
	return session
}

func (sm *SessionManager) ListSessions(projectPath string) []*Session {
	files, err := filepath.Glob(filepath.Join(sm.sessionsDir, "*.json"))
	if err != nil {
		return nil
	}

	var list []*Session
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}
		if filepath.Base(session.ProjectPath) == projectPath {
			sm.initRuntimeFields(&session)
			list = append(list, &session)
		}
	}
	return list
}

func (sm *SessionManager) ListSessionsByIDs(ids []string) []*Session {
	var list []*Session
	for _, id := range ids {
		session, err := sm.loadFromFile(id)
		if err != nil {
			continue
		}
		list = append(list, session)
	}
	return list
}

func (sm *SessionManager) SubmitInput(sessionID, input, mode, model string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
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
	session.Mode = mode
	if model != "" {
		session.Model = model
	}
	session.PermissionMgr.SetMode(mode)
	session.State = newState
	session.CreatedAt = time.Now()
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

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	if err := ValidateTransition(session.State, newState); err != nil {
		return err
	}

	oldState := session.State
	session.Mode = mode
	session.State = newState
	session.CreatedAt = time.Now()
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

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
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
	session.CreatedAt = time.Now()
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

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
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
	session.CreatedAt = time.Now()
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

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	filtered := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == openai.ChatMessageRoleUser && strings.Contains(msg.Content, "<reminder>") {
			continue
		}
		if msg.Role == openai.ChatMessageRoleTool && strings.Contains(msg.Content, "[Previous:") {
			continue
		}
		filtered = append(filtered, msg)
	}
	session.Messages = filtered
	session.CreatedAt = time.Now()
	sm.save(session)
	return nil
}

func (sm *SessionManager) save(session *Session) {
	path := filepath.Join(sm.sessionsDir, session.ID+".json")
	data, _ := json.MarshalIndent(session, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (sm *SessionManager) initRuntimeFields(session *Session) {
	sm.runtimeMu.Lock()
	defer sm.runtimeMu.Unlock()

	if rt, exists := sm.runtimeMap[session.ID]; exists {
		session.Ctx = rt.Ctx
		session.CancelFunc = rt.CancelFunc
		session.PermissionMgr = rt.PermissionMgr
		session.BlockedResponse = rt.BlockedResponse
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	mode := session.Mode
	if mode == "" {
		mode = security.PlanMode
	}
	pm := security.NewPermissionManager(mode, session.ProjectPath)
	blockingChan := make(chan security.BlockingRequest, 10)
	pm.SetBlockingChannel(blockingChan)

	rt := &SessionRuntime{
		Ctx:             ctx,
		CancelFunc:      cancel,
		PermissionMgr:   pm,
		BlockedResponse: make(chan PermissionDecision, 1),
	}

	sm.runtimeMap[session.ID] = rt

	session.Ctx = rt.Ctx
	session.CancelFunc = rt.CancelFunc
	session.PermissionMgr = rt.PermissionMgr
	session.BlockedResponse = rt.BlockedResponse
}

func (sm *SessionManager) loadFromFile(id string) (*Session, error) {
	path := filepath.Join(sm.sessionsDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	sm.initRuntimeFields(&session)
	return &session, nil
}

func (sm *SessionManager) cleanupRuntime(sessionID string) {
	sm.runtimeMu.Lock()
	defer sm.runtimeMu.Unlock()
	if rt, exists := sm.runtimeMap[sessionID]; exists {
		if rt.CancelFunc != nil {
			rt.CancelFunc()
		}
		delete(sm.runtimeMap, sessionID)
	}
}

func (sm *SessionManager) CompleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	if err := ValidateTransition(session.State, StateCompleted); err != nil {
		return err
	}

	oldState := session.State
	session.State = StateCompleted
	session.CreatedAt = time.Now()
	sm.save(session)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventCompleted, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"old_state":  oldState,
		}))
	}

	sm.cleanupRuntime(session.ID)

	return nil
}

func (sm *SessionManager) CancelSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, err := sm.loadFromFile(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	sm.cleanupRuntime(session.ID)

	if sm.eventBus != nil {
		sm.eventBus.Publish(session.ID, events.NewEvent(events.EventStateChange, session.ID, map[string]interface{}{
			"session_id": session.ID,
			"state":      "cancelled",
		}))
	}

	return nil
}

func (sm *SessionManager) SubmitPermissionDecision(sessionID string, decision PermissionDecision) error {
	session, err := sm.loadFromFile(sessionID)
	if err != nil {
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
