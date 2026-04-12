package api

import (
	"encoding/json"
	"sync"
	"time"

	"agent-base/internal/session"
	"agent-base/pkg/api"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

type WSClient struct {
	conn       *websocket.Conn
	sessionIDs []string
	send       chan api.WSEvent
	mu         sync.RWMutex
}

func NewWSClient(conn *websocket.Conn) *WSClient {
	return &WSClient{
		conn:       conn,
		sessionIDs: make([]string, 0),
		send:       make(chan api.WSEvent, 256),
	}
}

func (c *WSClient) Subscribe(sessionIDs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	existing := make(map[string]bool)
	for _, id := range c.sessionIDs {
		existing[id] = true
	}

	for _, id := range sessionIDs {
		if !existing[id] {
			c.sessionIDs = append(c.sessionIDs, id)
		}
	}
}

func (c *WSClient) IsSubscribed(sessionID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, id := range c.sessionIDs {
		if id == sessionID {
			return true
		}
	}
	return false
}

func (c *WSClient) Send(event api.WSEvent) {
	select {
	case c.send <- event:
	default:
	}
}

func (c *WSClient) Close() error {
	return c.conn.Close()
}

type WSManager struct {
	clients    map[*WSClient]bool
	mu         sync.RWMutex
	sessionMgr *session.SessionManager
}

func NewWSManager(sessionMgr *session.SessionManager) *WSManager {
	m := &WSManager{
		clients:    make(map[*WSClient]bool),
		sessionMgr: sessionMgr,
	}
	m.startEventForwarding()
	return m
}

func NewWSManagerWithSessionMgrInterface(listSessions func() []api.SessionInfo, subscribe func(string) chan session.SessionEvent, sendMessage func(string, string) error, sendPermissionResponse func(string, string, bool) error, sendControl func(string, string) error) *WSManager {
	m := &WSManager{
		clients: make(map[*WSClient]bool),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			sessions := listSessions()
			for _, sess := range sessions {
				ch := subscribe(sess.ID)
				go m.forwardSessionEvents(sess.ID, ch)
			}
		}
	}()

	return m
}

func (m *WSManager) AddClient(client *WSClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client] = true
}

func (m *WSManager) RemoveClient(client *WSClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, client)
}

func (m *WSManager) Broadcast(event api.WSEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for client := range m.clients {
		if client.IsSubscribed(event.SessionID) {
			client.Send(event)
		}
	}
}

func (m *WSManager) startEventForwarding() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			sessions := m.sessionMgr.ListSessions()
			for _, sess := range sessions {
				ch := m.sessionMgr.Subscribe(sess.ID)
				go m.forwardSessionEvents(sess.ID, ch)
			}
		}
	}()
}

func (m *WSManager) forwardSessionEvents(sessionID string, ch chan session.SessionEvent) {
	for event := range ch {
		wsEvent := api.WSEvent{
			Type:      string(event.Type),
			SessionID: sessionID,
			Timestamp: event.Timestamp,
			Data:      event.Data,
		}
		m.Broadcast(wsEvent)
	}
}

func (m *WSManager) HandleConnection(conn *websocket.Conn) {
	client := NewWSClient(conn)
	m.AddClient(client)

	defer func() {
		m.RemoveClient(client)
		client.Close()
		close(client.send)
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go m.writePump(client)
	m.readPump(client)
}

func (m *WSManager) readPump(client *WSClient) {
	defer func() {
		m.RemoveClient(client)
		client.Close()
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}
			break
		}

		var msg api.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		m.handleWSMessage(client, msg)
	}
}

func (m *WSManager) handleWSMessage(client *WSClient, msg api.WSMessage) {
	switch msg.Type {
	case "subscribe":
		client.Subscribe(msg.SessionIDs)
	case "send_message":
		if msg.SessionID != "" && msg.Content != "" {
			m.sessionMgr.SendMessage(msg.SessionID, msg.Content)
		}
	case "approve":
		if msg.SessionID != "" && msg.RequestID != "" {
			m.sessionMgr.SendPermissionResponse(msg.SessionID, msg.RequestID, true)
		}
	case "reject":
		if msg.SessionID != "" && msg.RequestID != "" {
			m.sessionMgr.SendPermissionResponse(msg.SessionID, msg.RequestID, false)
		}
	case "control":
		if msg.SessionID != "" {
			action := msg.Content
			if action == "" {
				action = "start"
			}
			m.sessionMgr.SendControl(msg.SessionID, action)
		}
	}
}

func (m *WSManager) writePump(client *WSClient) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Close()
	}()

	for {
		select {
		case event, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(data)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
