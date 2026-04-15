package websocket

type WSMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

const (
	WSCreateSession   = "create_session"
	WSSubmitInput     = "submit_input"
	WSApprovePlan     = "approve_plan"
	WSUnblockSession  = "unblock_session"
	WSGetSessionState = "get_session_state"
	WSListSessions    = "list_sessions"

	WSSessionCreated   = "session_created"
	WSStateUpdate      = "state_update"
	WSAssistantMessage = "assistant_message"
	WSToolExecution    = "tool_execution"
	WSBlocked          = "blocked"
	WSCompleted        = "completed"
	WSError            = "error"
)
