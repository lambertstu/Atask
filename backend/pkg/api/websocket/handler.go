package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"agent-base/internal/engine"
	"agent-base/internal/systems/project"
	"agent-base/internal/systems/session"
	"agent-base/pkg/security"

	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Handler struct {
	projectManager *project.ProjectManager
	sessionManager *session.SessionManager
	engine         *engine.AgentEngine
	permissionMgr  *security.PermissionManager
	clients        map[string]*websocket.Conn
	mu             sync.RWMutex
	currentSession string
}

func NewHandler(pm *project.ProjectManager, engine *engine.AgentEngine, permMgr *security.PermissionManager) *Handler {
	return &Handler{
		projectManager: pm,
		engine:         engine,
		permissionMgr:  permMgr,
		clients:        make(map[string]*websocket.Conn),
	}
}

func (h *Handler) SetSessionManager(sm *session.SessionManager) {
	h.sessionManager = sm
}

func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read message error: %v", err)
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			h.sendError(conn, "Invalid message format")
			continue
		}

		h.handleMessage(conn, msg)
	}
}

func (h *Handler) handleMessage(conn *websocket.Conn, msg WSMessage) {
	switch msg.Type {
	case WSCreateSession:
		h.handleCreateSession(conn, msg)
	case WSSubmitInput:
		h.handleSubmitInput(conn, msg)
	case WSApprovePlan:
		h.handleApprovePlan(conn, msg)
	case WSUnblockSession:
		h.handleUnblockSession(conn, msg)
	case WSGetSessionState:
		h.handleGetSessionState(conn, msg)
	case WSListSessions:
		h.handleListSessions(conn, msg)
	default:
		h.sendError(conn, "Unknown message type: "+msg.Type)
	}
}

func (h *Handler) handleCreateSession(conn *websocket.Conn, msg WSMessage) {
	projectPath, ok := msg.Data["project_path"].(string)
	if !ok {
		h.sendError(conn, "project_path required")
		return
	}

	model, ok := msg.Data["model"].(string)
	if !ok {
		model = "glm-5"
	}

	session := h.sessionManager.CreateSession(projectPath, model)
	h.projectManager.AddSession(projectPath, session.ID)

	h.mu.Lock()
	h.clients[session.ID] = conn
	h.mu.Unlock()

	h.send(conn, WSSessionCreated, map[string]interface{}{
		"session_id": session.ID,
		"state":      session.State,
	})
}

func (h *Handler) handleSubmitInput(conn *websocket.Conn, msg WSMessage) {
	sessionID, ok := msg.Data["session_id"].(string)
	if !ok {
		h.sendError(conn, "session_id required")
		return
	}

	input, ok := msg.Data["input"].(string)
	if !ok {
		h.sendError(conn, "input required")
		return
	}

	mode, _ := msg.Data["mode"].(string)
	if mode == "" {
		mode = "plan"
	}

	if err := h.sessionManager.SubmitInput(sessionID, input, mode); err != nil {
		h.sendError(conn, err.Error())
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	h.send(conn, WSStateUpdate, map[string]interface{}{
		"session_id": sessionID,
		"state":      sess.State,
	})

	h.permissionMgr.SetMode(mode)
	go h.runAgent(sess)
}

func (h *Handler) handleApprovePlan(conn *websocket.Conn, msg WSMessage) {
	sessionID, ok := msg.Data["session_id"].(string)
	if !ok {
		h.sendError(conn, "session_id required")
		return
	}

	if err := h.sessionManager.Transition(sessionID, session.StateProcessing); err != nil {
		h.sendError(conn, err.Error())
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	h.send(conn, WSStateUpdate, map[string]interface{}{
		"session_id": sessionID,
		"state":      sess.State,
	})

	h.permissionMgr.SetMode("build")
	go h.runAgent(sess)
}

func (h *Handler) handleUnblockSession(conn *websocket.Conn, msg WSMessage) {
	sessionID, ok := msg.Data["session_id"].(string)
	if !ok {
		h.sendError(conn, "session_id required")
		return
	}

	response, ok := msg.Data["response"].(string)
	if !ok {
		h.sendError(conn, "response required")
		return
	}

	if err := h.sessionManager.Unblock(sessionID, response); err != nil {
		h.sendError(conn, err.Error())
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	h.send(conn, WSStateUpdate, map[string]interface{}{
		"session_id": sessionID,
		"state":      sess.State,
	})

	go h.runAgent(sess)
}

func (h *Handler) handleGetSessionState(conn *websocket.Conn, msg WSMessage) {
	sessionID, ok := msg.Data["session_id"].(string)
	if !ok {
		h.sendError(conn, "session_id required")
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	if sess == nil {
		h.sendError(conn, "session not found")
		return
	}

	h.send(conn, WSStateUpdate, map[string]interface{}{
		"session_id": sessionID,
		"state":      sess.State,
		"blocked_on": sess.BlockedOn,
	})
}

func (h *Handler) handleListSessions(conn *websocket.Conn, msg WSMessage) {
	projectPath, ok := msg.Data["project_path"].(string)
	if !ok {
		h.sendError(conn, "project_path required")
		return
	}

	sessions := h.sessionManager.ListSessions(projectPath)
	var list []map[string]interface{}
	for _, s := range sessions {
		list = append(list, map[string]interface{}{
			"id":         s.ID,
			"state":      s.State,
			"created_at": s.CreatedAt,
		})
	}

	h.send(conn, WSStateUpdate, map[string]interface{}{
		"sessions": list,
	})
}

func (h *Handler) runAgent(sess *session.Session) {
	conn := h.getClient(sess.ID)
	if conn == nil {
		return
	}

	h.mu.Lock()
	h.currentSession = sess.ID
	h.mu.Unlock()

	h.permissionMgr.SetBlockedCallback(func(toolName string, toolInput map[string]interface{}) {
		h.mu.RLock()
		currentID := h.currentSession
		h.mu.RUnlock()

		if currentID == sess.ID {
			h.sessionManager.SetBlocked(sess.ID, "permission", toolName, toolInput)
		}
	})

	ctx := context.Background()
	messages := sess.Messages

	for {
		sess = h.sessionManager.GetSession(sess.ID)
		if sess == nil || sess.State == session.StateBlocked {
			h.send(conn, WSBlocked, map[string]interface{}{
				"session_id":   sess.ID,
				"blocked_on":   sess.BlockedOn,
				"blocked_tool": sess.BlockedTool,
			})
			return
		}

		if sess.State == session.StateCompleted {
			h.send(conn, WSCompleted, map[string]interface{}{
				"session_id": sess.ID,
			})
			return
		}

		result, err := h.engine.Run(ctx, messages)
		if err != nil {
			h.sendError(conn, fmt.Sprintf("Agent error: %v", err))
			return
		}

		messages = result
		h.sessionManager.UpdateMessages(sess.ID, result)

		lastMsg := result[len(result)-1]
		if lastMsg.Role == openai.ChatMessageRoleAssistant && lastMsg.Content != "" {
			h.send(conn, WSAssistantMessage, map[string]interface{}{
				"session_id": sess.ID,
				"content":    lastMsg.Content,
			})
		}

		if isCompleted(result) {
			h.sessionManager.Transition(sess.ID, session.StateCompleted)
			h.send(conn, WSCompleted, map[string]interface{}{
				"session_id": sess.ID,
			})
			return
		}
	}
}

func isCompleted(messages []openai.ChatCompletionMessage) bool {
	if len(messages) == 0 {
		return false
	}
	lastMsg := messages[len(messages)-1]
	return lastMsg.Role == openai.ChatMessageRoleAssistant &&
		len(lastMsg.ToolCalls) == 0 &&
		lastMsg.Content != ""
}

func (h *Handler) getClient(sessionID string) *websocket.Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clients[sessionID]
}

func (h *Handler) send(conn *websocket.Conn, msgType string, data map[string]interface{}) {
	msg := WSMessage{
		Type: msgType,
		Data: data,
	}
	msgBytes, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, msgBytes)
}

func (h *Handler) sendError(conn *websocket.Conn, errMsg string) {
	h.send(conn, WSError, map[string]interface{}{
		"error": errMsg,
	})
}
