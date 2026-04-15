package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

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
}

type SessionManager struct {
	sessionsDir string
	sessions    map[string]*Session
	mu          sync.RWMutex
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func NewSessionManager(projectPath string) *SessionManager {
	sessionsDir := filepath.Join(projectPath, ".sessions")
	os.MkdirAll(sessionsDir, 0755)

	sm := &SessionManager{
		sessionsDir: sessionsDir,
		sessions:    make(map[string]*Session),
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
		sm.sessions[session.ID] = &session
	}
}

func (sm *SessionManager) CreateSession(projectPath, model string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:          generateID(),
		ProjectPath: projectPath,
		Model:       model,
		State:       StatePending,
		CreatedAt:   time.Now(),
		Messages:    []openai.ChatCompletionMessage{},
	}
	sm.sessions[session.ID] = session
	sm.save(session)
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
		if s.ProjectPath == projectPath {
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

	session.State = newState
	sm.save(session)
	return nil
}

func (sm *SessionManager) Transition(sessionID string, newState SessionState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}

	if err := ValidateTransition(session.State, newState); err != nil {
		return err
	}

	session.State = newState
	sm.save(session)
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

	session.State = StateBlocked
	session.BlockedOn = blockedOn
	session.BlockedTool = blockedTool
	session.BlockedArgs = blockedArgs
	sm.save(session)
	return nil
}

func (sm *SessionManager) Unblock(sessionID, response string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := sm.sessions[sessionID]
	if session == nil {
		return fmt.Errorf("session not found")
	}
	if session.State != StateBlocked {
		return fmt.Errorf("session not in blocked state")
	}

	session.Messages = append(session.Messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: response,
	})
	session.State = StateProcessing
	session.BlockedOn = ""
	session.BlockedTool = ""
	session.BlockedArgs = nil
	sm.save(session)
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
