package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"agent-base/internal/engine"
	"agent-base/internal/systems/session"
	"agent-base/pkg/events"
	"agent-base/pkg/security"
)

type CreateSessionRequest struct {
	ProjectPath string `json:"project_path"`
	Model       string `json:"model"`
}

type SessionResponse struct {
	ID          string `json:"id"`
	ProjectPath string `json:"project_path"`
	Model       string `json:"model"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	BlockedOn   string `json:"blocked_on,omitempty"`
	BlockedTool string `json:"blocked_tool,omitempty"`
}

type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

type SubmitInputRequest struct {
	Input string `json:"input"`
	Mode  string `json:"mode"`
}

type UnblockRequest struct {
	Response   string `json:"response"`
	Approved   bool   `json:"approved"`
	AddAllowed string `json:"add_allowed"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Handler struct {
	sessionManager *session.SessionManager
	engine         *engine.AgentEngine
	eventBus       *events.EventBus
}

func NewHandler(sm *session.SessionManager, eng *engine.AgentEngine, eb *events.EventBus) *Handler {
	return &Handler{
		sessionManager: sm,
		engine:         eng,
		eventBus:       eb,
	}
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "glm-5"
	}

	sess := h.sessionManager.CreateSession(req.ProjectPath, model)

	sendJSON(w, SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, http.StatusOK)
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	projectPath := r.URL.Query().Get("project_path")
	if projectPath == "" {
		sendError(w, "project_path required", http.StatusBadRequest)
		return
	}

	sessions := h.sessionManager.ListSessions(projectPath)
	var list []SessionResponse
	for _, s := range sessions {
		list = append(list, SessionResponse{
			ID:          s.ID,
			ProjectPath: s.ProjectPath,
			Model:       s.Model,
			State:       string(s.State),
			CreatedAt:   s.CreatedAt.Format(time.RFC3339),
			BlockedOn:   s.BlockedOn,
			BlockedTool: s.BlockedTool,
		})
	}

	sendJSON(w, SessionListResponse{Sessions: list}, http.StatusOK)
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "id")
	if sessionID == "" {
		sendError(w, "session_id required", http.StatusBadRequest)
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	if sess == nil {
		sendError(w, "session not found", http.StatusNotFound)
		return
	}

	sendJSON(w, SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
		BlockedOn:   sess.BlockedOn,
		BlockedTool: sess.BlockedTool,
	}, http.StatusOK)
}

func (h *Handler) SubmitInput(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "id")
	if sessionID == "" {
		sendError(w, "session_id required", http.StatusBadRequest)
		return
	}

	var req SubmitInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "plan"
	}

	if err := h.sessionManager.SubmitInput(sessionID, req.Input, mode); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	if sess == nil {
		sendError(w, "session not found", http.StatusNotFound)
		return
	}

	if sess.PermissionMgr != nil {
		sess.PermissionMgr.SetMode(mode)
	}

	go h.runAgent(sess)

	sendJSON(w, SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, http.StatusOK)
}

func (h *Handler) ApprovePlan(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "id")
	if sessionID == "" {
		sendError(w, "session_id required", http.StatusBadRequest)
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	if sess == nil {
		sendError(w, "session not found", http.StatusNotFound)
		return
	}

	if err := h.sessionManager.Transition(sessionID, session.StateProcessing); err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if sess.PermissionMgr != nil {
		sess.PermissionMgr.SetMode("build")
	}

	go h.runAgent(sess)

	sendJSON(w, SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, http.StatusOK)
}

func (h *Handler) UnblockSession(w http.ResponseWriter, r *http.Request) {
	sessionID := getPathParam(r, "id")
	if sessionID == "" {
		sendError(w, "session_id required", http.StatusBadRequest)
		return
	}

	var req UnblockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sess := h.sessionManager.GetSession(sessionID)
	if sess == nil {
		sendError(w, "session not found", http.StatusNotFound)
		return
	}

	if sess.State != session.StateBlocked {
		sendError(w, "session not in blocked state", http.StatusBadRequest)
		return
	}

	if sess.PermissionMgr != nil && sess.PermissionMgr.IsBlockingMode() {
		blockingChan := sess.PermissionMgr.GetBlockingChannel()
		if blockingChan != nil {
			select {
			case pendingReq := <-blockingChan:
				pendingReq.ResponseCh <- security.BlockingResponse{
					Approved:   req.Approved,
					AddAllowed: req.AddAllowed,
				}
			default:
			}
		}
	}

	if req.Approved {
		if err := h.sessionManager.Unblock(sessionID, req.Response); err != nil {
			sendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		go h.runAgent(sess)
	} else {
		h.sessionManager.Transition(sessionID, session.StateCompleted)
	}

	sess = h.sessionManager.GetSession(sessionID)
	sendJSON(w, SessionResponse{
		ID:          sess.ID,
		ProjectPath: sess.ProjectPath,
		Model:       sess.Model,
		State:       string(sess.State),
		CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
	}, http.StatusOK)
}

func (h *Handler) runAgent(sess *session.Session) {
	if sess.Ctx == nil || sess.CancelFunc == nil {
		ctx, cancel := context.WithCancel(context.Background())
		sess.Ctx = ctx
		sess.CancelFunc = cancel
	}

	emitter := &SessionEmitter{
		eventBus:  h.eventBus,
		sessionID: sess.ID,
	}

	_, err := h.engine.RunStream(sess.Ctx, sess.Messages, emitter, sess.ID)
	if err != nil {
		h.eventBus.Publish(sess.ID, events.NewEvent(events.EventError, sess.ID, map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	h.sessionManager.CompleteSession(sess.ID)
}

type SessionEmitter struct {
	eventBus  *events.EventBus
	sessionID string
}

func (e *SessionEmitter) Emit(eventType events.EventType, data map[string]interface{}) {
	if e.eventBus != nil {
		e.eventBus.Publish(e.sessionID, events.NewEvent(eventType, e.sessionID, data))
	}
}

func sendJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, message string, status int) {
	sendJSON(w, ErrorResponse{Error: message}, status)
}

func getPathParam(r *http.Request, name string) string {
	path := r.URL.Path
	parts := splitPath(path)
	for i, part := range parts {
		if part == name && i+1 < len(parts) {
			return parts[i+1]
		}
		if part != "sessions" && part != "api" && part != "" {
			return part
		}
	}
	return ""
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range split(path, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func split(s, sep string) []string {
	var result []string
	for {
		idx := indexOf(s, sep)
		if idx == -1 {
			if s != "" {
				result = append(result, s)
			}
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOf(s, sep string) int {
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func contextWithCancel() (interface{}, interface{}) {
	return nil, nil
}
