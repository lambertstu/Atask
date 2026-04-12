package api

import "encoding/json"

type SessionStatus string

const (
	StatusPlanning     SessionStatus = "planning"
	StatusScheduled    SessionStatus = "scheduled"
	StatusInProcessing SessionStatus = "in_processing"
	StatusHumanReview  SessionStatus = "human_review"
	StatusCompleted    SessionStatus = "completed"
)

func (s SessionStatus) String() string {
	return string(s)
}

func (s SessionStatus) DisplayName() string {
	switch s {
	case StatusPlanning:
		return "Planning"
	case StatusScheduled:
		return "Scheduled"
	case StatusInProcessing:
		return "In Progress"
	case StatusHumanReview:
		return "Human Review"
	case StatusCompleted:
		return "Completed"
	default:
		return string(s)
	}
}

type SessionInfo struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	WorkDir     string        `json:"work_dir"`
	Status      SessionStatus `json:"status"`
	CreatedAt   int64         `json:"created_at"`
	LastActive  int64         `json:"last_active"`
	Description string        `json:"description,omitempty"`
}

type CreateSessionRequest struct {
	Name    string `json:"name"`
	WorkDir string `json:"work_dir,omitempty"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type ControlRequest struct {
	Action string `json:"action"`
}

type PermissionResponseRequest struct {
	RequestID string `json:"request_id"`
}

type WSEvent struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type WSMessage struct {
	Type       string   `json:"type"`
	SessionID  string   `json:"session_id,omitempty"`
	SessionIDs []string `json:"session_ids,omitempty"`
	Content    string   `json:"content,omitempty"`
	RequestID  string   `json:"request_id,omitempty"`
	Approved   bool     `json:"approved,omitempty"`
}

func (e *WSEvent) UnmarshalJSON(data []byte) error {
	type Alias WSEvent
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Data == nil {
		e.Data = make(map[string]interface{})
	}
	return nil
}
