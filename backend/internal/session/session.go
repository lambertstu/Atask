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
	"agent-base/internal/engine"
	"agent-base/internal/llm"
	"agent-base/internal/systems/memory"
	"agent-base/internal/systems/skills"
	"agent-base/internal/systems/subagent"
	"agent-base/internal/systems/tasks"
	"agent-base/internal/tools"
	"agent-base/pkg/events"
	"agent-base/pkg/security"

	"github.com/sashabaranov/go-openai"
)

type InputType string

const (
	InputUserMessage   InputType = "user_message"
	InputPermissionRes InputType = "permission_response"
	InputControl       InputType = "control"
)

type InputMessage struct {
	Type    InputType
	Content string
	Data    map[string]interface{}
}

type PermissionRequest struct {
	RequestID string
	Approved  bool
}

type Session struct {
	ID            string
	Name          string
	WorkDir       string
	SessionDir    string
	Status        SessionStatus
	StatusHistory []StatusTransition
	CreatedAt     int64
	LastActive    int64

	History       []openai.ChatCompletionMessage
	Registry      *tools.DefaultRegistry
	MemoryMgr     *memory.MemoryManager
	TaskMgr       *tasks.TaskManager
	BackgroundMgr *tasks.BackgroundManager
	CronScheduler *tasks.CronScheduler
	SkillLoader   *skills.SkillLoader
	PermissionMgr *security.PermissionManager
	HookMgr       *events.HookManager
	PromptBuilder engine.PromptBuilder
	ContextMgr    engine.ContextManager
	RecoveryMgr   engine.RecoveryManager
	AgentEngine   *engine.AgentEngine

	Config    *config.Config
	LLMClient llm.LLMClient
	EventBus  *EventBus
	InputChan chan InputMessage
	Ctx       context.Context
	Cancel    context.CancelFunc

	mu           sync.RWMutex
	pendingPerms map[string]chan PermissionRequest
}

func NewSession(id, name, workDir string, cfg *config.Config, llmClient llm.LLMClient, globalRegistry *tools.DefaultRegistry, eventBus *EventBus) (*Session, error) {
	sessionDir := filepath.Join(workDir, ".sessions", id)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Session{
		ID:            id,
		Name:          name,
		WorkDir:       workDir,
		SessionDir:    sessionDir,
		Status:        StatusPlanning,
		StatusHistory: []StatusTransition{},
		CreatedAt:     time.Now().Unix(),
		LastActive:    time.Now().Unix(),
		Config:        cfg,
		LLMClient:     llmClient,
		EventBus:      eventBus,
		InputChan:     make(chan InputMessage, 100),
		Ctx:           ctx,
		Cancel:        cancel,
		pendingPerms:  make(map[string]chan PermissionRequest),
		History:       []openai.ChatCompletionMessage{},
	}

	s.Registry = globalRegistry.Clone()

	memoryDir := filepath.Join(sessionDir, ".memory")
	s.MemoryMgr = memory.NewMemoryManager(memoryDir)
	s.MemoryMgr.LoadAll()

	tasksDir := filepath.Join(sessionDir, ".tasks")
	s.TaskMgr = tasks.NewTaskManager(tasksDir)

	backgroundDir := filepath.Join(sessionDir, ".runtime-tasks")
	s.BackgroundMgr = tasks.NewBackgroundManagerWithWorkDir(backgroundDir, workDir)

	s.CronScheduler = tasks.NewCronScheduler()

	skillsDir := filepath.Join(workDir, "skills")
	s.SkillLoader = skills.NewSkillLoader(skillsDir)
	s.SkillLoader.LoadAll()

	s.PermissionMgr = security.NewPermissionManager("plan", workDir)

	s.HookMgr = events.NewHookManager(workDir, false)

	s.PromptBuilder = engine.NewSystemPromptBuilder(workDir, cfg.Model)

	s.ContextMgr = engine.NewContextManager(llmClient, cfg.Model, sessionDir, cfg.ContextThreshold)

	s.RecoveryMgr = engine.NewRecoveryManager(llmClient, cfg.Model, s.ContextMgr, s.PromptBuilder)

	s.AgentEngine = engine.NewAgentEngine(
		llmClient,
		s.Registry,
		s.PermissionMgr,
		s.HookMgr,
		s.PromptBuilder,
		s.ContextMgr,
		s.RecoveryMgr,
		cfg.Model,
		cfg.ContextThreshold,
	)

	s.AgentEngine.SetSessionInfo(s.ID, workDir, s.handlePermission, s.publishEventCallback)

	s.registerSessionTools()

	s.saveMeta()

	return s, nil
}

func (s *Session) registerSessionTools() {
	s.Registry.Register(tasks.NewTaskCreateTool(s.TaskMgr))
	s.Registry.Register(tasks.NewTaskUpdateTool(s.TaskMgr))
	s.Registry.Register(tasks.NewTaskListTool(s.TaskMgr))
	s.Registry.Register(tasks.NewTaskGetTool(s.TaskMgr))
	s.Registry.Register(tasks.NewBackgroundRunTool(s.BackgroundMgr))
	s.Registry.Register(tasks.NewCheckBackgroundTool(s.BackgroundMgr))
	s.Registry.Register(tasks.NewCronCreateTool(s.CronScheduler))
	s.Registry.Register(tasks.NewCronDeleteTool(s.CronScheduler))
	s.Registry.Register(tasks.NewCronListTool(s.CronScheduler))
	s.Registry.Register(memory.NewSaveMemoryTool(s.MemoryMgr))
	s.Registry.Register(skills.NewLoadSkillTool(s.SkillLoader))
	s.Registry.Register(subagent.NewTaskTool(subagent.NewSubagentRunner(s.LLMClient, s.Registry, s.WorkDir, s.Config.Model)))
	s.Registry.Register(engine.NewCompactTool())
}

func (s *Session) Run() {
	s.CronScheduler.Start()
	defer s.CronScheduler.Stop()

	s.publishEvent(EventStatusChange, map[string]interface{}{
		"from":   StatusPlanning,
		"to":     s.Status,
		"reason": "session started",
	})

	for {
		select {
		case <-s.Ctx.Done():
			s.transitionStatus(StatusCompleted, "session stopped", "")
			return
		case msg := <-s.InputChan:
			s.handleInput(msg)
		}
	}
}

func (s *Session) handleInput(msg InputMessage) {
	s.mu.Lock()
	s.LastActive = time.Now().Unix()
	s.mu.Unlock()

	switch msg.Type {
	case InputControl:
		s.handleControl(msg.Content, msg.Data)
	case InputUserMessage:
		s.handleUserMessage(msg.Content)
	case InputPermissionRes:
		s.handlePermissionResponse(msg.Data)
	}
}

func (s *Session) handleControl(command string, data map[string]interface{}) {
	switch command {
	case "start":
		if s.Status == StatusPlanning || s.Status == StatusScheduled {
			s.transitionStatus(StatusInProcessing, "user started session", "")
		}
	case "pause":
		if s.Status == StatusInProcessing {
			s.transitionStatus(StatusScheduled, "user paused session", "")
		}
	case "cancel":
		s.transitionStatus(StatusCompleted, "user cancelled session", "")
		s.Cancel()
	case "schedule":
		if s.Status == StatusPlanning {
			s.transitionStatus(StatusScheduled, "user scheduled session", "")
		}
	}
}

func (s *Session) handleUserMessage(content string) {
	if s.Status != StatusScheduled && s.Status != StatusInProcessing {
		s.publishEvent(EventError, map[string]interface{}{
			"error": "session not ready to process messages",
		})
		return
	}

	if s.Status == StatusScheduled {
		s.transitionStatus(StatusInProcessing, "user message received", "")
	}

	bgNotifs := s.BackgroundMgr.DrainNotifications()
	for _, notif := range bgNotifs {
		s.publishEvent(EventOutput, map[string]interface{}{
			"content": fmt.Sprintf("[bg:%s] %s: %s", notif.TaskID, notif.Status, notif.Preview),
			"type":    "background",
		})
		s.History = append(s.History, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("<background-results>\n[bg:%s] %s: %s (output_file=%s)\n</background-results>", notif.TaskID, notif.Status, notif.Preview, notif.OutputFile),
		})
	}

	cronNotifs := s.CronScheduler.DrainNotifications()
	for _, notif := range cronNotifs {
		s.publishEvent(EventOutput, map[string]interface{}{
			"content": fmt.Sprintf("[Cron] %s", notif),
			"type":    "cron",
		})
		s.History = append(s.History, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: notif,
		})
	}

	s.History = append(s.History, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	})

	newHistory, err := s.AgentEngine.Run(s.Ctx, s.History)
	if err != nil {
		s.publishEvent(EventError, map[string]interface{}{
			"error": err.Error(),
		})
		s.transitionStatus(StatusScheduled, "error occurred", "")
		return
	}

	s.History = newHistory
	s.saveHistory()

	if len(s.History) > 0 {
		lastMsg := s.History[len(s.History)-1]
		if lastMsg.Role == openai.ChatMessageRoleAssistant && lastMsg.Content != "" {
			s.publishEvent(EventOutput, map[string]interface{}{
				"content":  lastMsg.Content,
				"is_final": len(lastMsg.ToolCalls) == 0,
			})
		}

		if len(lastMsg.ToolCalls) == 0 {
			s.transitionStatus(StatusCompleted, "agent finished", "")
		}
	}
}

func (s *Session) handlePermissionResponse(data map[string]interface{}) {
	requestID, ok := data["request_id"].(string)
	if !ok {
		return
	}

	approved, ok := data["approved"].(bool)
	if !ok {
		return
	}

	s.mu.Lock()
	if ch, exists := s.pendingPerms[requestID]; exists {
		ch <- PermissionRequest{RequestID: requestID, Approved: approved}
		delete(s.pendingPerms, requestID)
	}
	s.mu.Unlock()

	if approved {
		s.transitionStatus(StatusInProcessing, "permission approved", "")
	} else {
		s.transitionStatus(StatusScheduled, "permission denied", "")
	}
}

func (s *Session) WaitForPermission(requestID string) bool {
	s.mu.Lock()
	ch := make(chan PermissionRequest, 1)
	s.pendingPerms[requestID] = ch
	s.mu.Unlock()

	s.transitionStatus(StatusHumanReview, "permission required", requestID)

	select {
	case resp := <-ch:
		return resp.Approved
	case <-s.Ctx.Done():
		return false
	case <-time.After(5 * time.Minute):
		s.mu.Lock()
		delete(s.pendingPerms, requestID)
		s.mu.Unlock()
		return false
	}
}

func (s *Session) transitionStatus(newStatus SessionStatus, reason, requestID string) {
	s.mu.Lock()
	oldStatus := s.Status
	s.Status = newStatus
	s.StatusHistory = append(s.StatusHistory, NewStatusTransition(oldStatus, newStatus, reason))
	s.mu.Unlock()

	s.publishEvent(EventStatusChange, map[string]interface{}{
		"from":       oldStatus,
		"to":         newStatus,
		"reason":     reason,
		"request_id": requestID,
	})

	s.saveMeta()
}

func (s *Session) publishEvent(eventType EventType, data map[string]interface{}) {
	s.EventBus.Publish(SessionEvent{
		SessionID: s.ID,
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	})
}

func (s *Session) saveMeta() {
	meta := map[string]interface{}{
		"id":             s.ID,
		"name":           s.Name,
		"work_dir":       s.WorkDir,
		"session_dir":    s.SessionDir,
		"status":         s.Status,
		"created_at":     s.CreatedAt,
		"last_active":    s.LastActive,
		"status_history": s.StatusHistory,
	}

	metaPath := filepath.Join(s.SessionDir, "meta.json")
	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, data, 0644)
}

func (s *Session) saveHistory() {
	historyPath := filepath.Join(s.SessionDir, "history.json")
	data, _ := json.MarshalIndent(s.History, "", "  ")
	os.WriteFile(historyPath, data, 0644)
}

func (s *Session) LoadHistory() error {
	historyPath := filepath.Join(s.SessionDir, "history.json")
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.History)
}

func (s *Session) Stop() {
	s.Cancel()
}

func (s *Session) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

func (s *Session) GetInfo() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("[%s] %s: %s (work_dir=%s)",
		StatusColor(s.Status)+StatusDisplayName(s.Status)+"\033[0m",
		s.ID,
		s.Name,
		s.WorkDir)
}

func (s *Session) GetStatusHistory() []StatusTransition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.StatusHistory
}

func (s *Session) handlePermission(requestID, toolName, reason string) bool {
	return s.WaitForPermission(requestID)
}

func (s *Session) publishEventCallback(eventType string, data map[string]interface{}) {
	s.EventBus.Publish(SessionEvent{
		SessionID: s.ID,
		Type:      EventType(eventType),
		Timestamp: time.Now().Unix(),
		Data:      data,
	})
}
