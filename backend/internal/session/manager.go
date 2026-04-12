package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"agent-base/internal/config"
	"agent-base/internal/llm"
	"agent-base/internal/tools"
)

type SessionManager struct {
	GlobalRegistry *tools.DefaultRegistry
	LLMClient      llm.LLMClient
	Config         *config.Config
	EventBus       *EventBus

	Sessions      map[string]*Session
	InputChannels map[string]chan InputMessage

	mu sync.RWMutex
}

func NewSessionManager(cfg *config.Config, llmClient llm.LLMClient, globalRegistry *tools.DefaultRegistry) *SessionManager {
	return &SessionManager{
		GlobalRegistry: globalRegistry,
		LLMClient:      llmClient,
		Config:         cfg,
		EventBus:       NewEventBus(),
		Sessions:       make(map[string]*Session),
		InputChannels:  make(map[string]chan InputMessage),
	}
}

func (m *SessionManager) NewSession(workDir, name string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := generateSessionID()

	session, err := NewSession(sessionID, name, workDir, m.Config, m.LLMClient, m.GlobalRegistry, m.EventBus)
	if err != nil {
		return nil, err
	}

	m.Sessions[sessionID] = session
	m.InputChannels[sessionID] = session.InputChan

	go session.Run()

	return session, nil
}

func (m *SessionManager) GetSession(sessionID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.Sessions[sessionID]
	return session, ok
}

func (m *SessionManager) SendInput(sessionID string, msg InputMessage) error {
	m.mu.RLock()
	ch, ok := m.InputChannels[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	select {
	case ch <- msg:
		return nil
	default:
		return fmt.Errorf("session %s input channel full", sessionID)
	}
}

func (m *SessionManager) SendMessage(sessionID, content string) error {
	return m.SendInput(sessionID, InputMessage{
		Type:    InputUserMessage,
		Content: content,
	})
}

func (m *SessionManager) SendControl(sessionID, command string) error {
	return m.SendInput(sessionID, InputMessage{
		Type:    InputControl,
		Content: command,
	})
}

func (m *SessionManager) SendPermissionResponse(sessionID, requestID string, approved bool) error {
	return m.SendInput(sessionID, InputMessage{
		Type: InputPermissionRes,
		Data: map[string]interface{}{
			"request_id": requestID,
			"approved":   approved,
		},
	})
}

func (m *SessionManager) ListSessions() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []SessionInfo
	for _, session := range m.Sessions {
		infos = append(infos, SessionInfo{
			ID:        session.ID,
			Name:      session.Name,
			Status:    session.GetStatus(),
			WorkDir:   session.WorkDir,
			CreatedAt: session.CreatedAt,
		})
	}
	return infos
}

func (m *SessionManager) StopSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.Sessions[sessionID]
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Stop()
	delete(m.Sessions, sessionID)
	delete(m.InputChannels, sessionID)

	return nil
}

func (m *SessionManager) RestoreSessions(workDir string) error {
	sessionsDir := filepath.Join(workDir, ".sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionDir := filepath.Join(sessionsDir, entry.Name())
		metaPath := filepath.Join(sessionDir, "meta.json")

		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		var meta map[string]interface{}
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}

		sessionID, _ := meta["id"].(string)
		name, _ := meta["name"].(string)
		sessionWorkDir, _ := meta["work_dir"].(string)

		session, err := NewSession(sessionID, name, sessionWorkDir, m.Config, m.LLMClient, m.GlobalRegistry, m.EventBus)
		if err != nil {
			continue
		}

		if err := session.LoadHistory(); err == nil {
		}

		m.mu.Lock()
		m.Sessions[sessionID] = session
		m.InputChannels[sessionID] = session.InputChan
		m.mu.Unlock()

		go session.Run()
	}

	return nil
}

func (m *SessionManager) Subscribe(sessionID string) chan SessionEvent {
	return m.EventBus.Subscribe(sessionID)
}

func (m *SessionManager) Unsubscribe(sessionID string, ch chan SessionEvent) {
	m.EventBus.Unsubscribe(sessionID, ch)
}

func (m *SessionManager) GetActiveSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []string
	for id, session := range m.Sessions {
		if session.GetStatus() == StatusInProcessing || session.GetStatus() == StatusHumanReview {
			active = append(active, id)
		}
	}
	return active
}

func (m *SessionManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, session := range m.Sessions {
		session.Stop()
	}

	<-ctx.Done()
}

type SessionInfo struct {
	ID        string
	Name      string
	Status    SessionStatus
	WorkDir   string
	CreatedAt int64
}

func (i SessionInfo) String() string {
	return fmt.Sprintf("[%s] %s: %s",
		StatusColor(i.Status)+StatusDisplayName(i.Status)+"\033[0m",
		i.ID,
		i.Name)
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano()/1000000)
}
