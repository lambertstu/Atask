package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-base/internal/config"
	"agent-base/internal/session"
	"agent-base/pkg/api"

	"github.com/gorilla/websocket"
)

type APIServer struct {
	sessionMgr *session.SessionManager
	wsManager  *WSManager
	mux        *http.ServeMux
}

func NewAPIServer(sessionMgr *session.SessionManager, cfg *config.Config) *APIServer {
	s := &APIServer{
		sessionMgr: sessionMgr,
		mux:        http.NewServeMux(),
	}
	s.wsManager = NewWSManager(sessionMgr)
	s.setupRoutes()
	return s
}

func (s *APIServer) setupRoutes() {
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/sessions/", s.handleSessionByID)
	s.mux.HandleFunc("/api/ws", s.HandleWebSocket)
	s.mux.HandleFunc("/health", s.handleHealth)
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *APIServer) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSessions(w, r)
	case http.MethodPost:
		s.createSession(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	sessionID := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.getSession(w, r, sessionID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	action := parts[1]
	switch action {
	case "messages":
		if r.Method == http.MethodPost {
			s.sendMessage(w, r, sessionID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "control":
		if r.Method == http.MethodPost {
			s.controlSession(w, r, sessionID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "approve":
		if r.Method == http.MethodPost {
			s.approvePermission(w, r, sessionID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	case "reject":
		if r.Method == http.MethodPost {
			s.rejectPermission(w, r, sessionID)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}

func (s *APIServer) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.sessionMgr.ListSessions()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *APIServer) createSession(w http.ResponseWriter, r *http.Request) {
	var req api.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Session name is required", http.StatusBadRequest)
		return
	}

	workDir := req.WorkDir
	if workDir == "" {
		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "Failed to load config", http.StatusInternalServerError)
			return
		}
		workDir = cfg.WorkDir
	}

	session, err := s.sessionMgr.NewSession(workDir, req.Name)
	if err != nil {
		http.Error(w, "Failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = session

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func (s *APIServer) getSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	session, ok := s.sessionMgr.GetSession(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	_ = session

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (s *APIServer) sendMessage(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req api.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	if err := s.sessionMgr.SendMessage(sessionID, req.Content); err != nil {
		http.Error(w, "Failed to send message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *APIServer) controlSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req api.ControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Action == "" {
		http.Error(w, "Action is required", http.StatusBadRequest)
		return
	}

	if err := s.sessionMgr.SendControl(sessionID, req.Action); err != nil {
		http.Error(w, "Failed to control session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *APIServer) approvePermission(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req api.PermissionResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RequestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	if err := s.sessionMgr.SendPermissionResponse(sessionID, req.RequestID, true); err != nil {
		http.Error(w, "Failed to approve: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *APIServer) rejectPermission(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req api.PermissionResponseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RequestID == "" {
		http.Error(w, "Request ID is required", http.StatusBadRequest)
		return
	}

	if err := s.sessionMgr.SendPermissionResponse(sessionID, req.RequestID, false); err != nil {
		http.Error(w, "Failed to reject: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *APIServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := UpgraderPool
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.wsManager.HandleConnection(conn)
}

func (s *APIServer) Mux() http.Handler {
	return s.mux
}

var UpgraderPool = websocketUpgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type websocketUpgrader struct {
	CheckOrigin func(*http.Request) bool
}

func (u websocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, respondHeader http.Header) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: u.CheckOrigin,
	}
	return upgrader.Upgrade(w, r, respondHeader)
}
