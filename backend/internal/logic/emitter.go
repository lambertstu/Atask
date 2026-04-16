package logic

import (
	"agent-base/internal/engine"
	"agent-base/internal/systems/session"
	"agent-base/pkg/events"
)

type SessionEmitter struct {
	eventBus  *events.EventBus
	sessionID string
}

func (e *SessionEmitter) Emit(eventType events.EventType, data map[string]interface{}) {
	if e.eventBus != nil {
		e.eventBus.Publish(e.sessionID, events.NewEvent(eventType, e.sessionID, data))
	}
}

func NewSessionEmitter(eventBus *events.EventBus, sessionID string) *SessionEmitter {
	return &SessionEmitter{
		eventBus:  eventBus,
		sessionID: sessionID,
	}
}

var _ engine.EventEmitter = (*SessionEmitter)(nil)

func RunAgent(eng *engine.AgentEngine, sm *session.SessionManager, eb *events.EventBus, sess *session.Session) {
	if eng == nil {
		eb.Publish(sess.ID, events.NewEvent(events.EventError, sess.ID, map[string]interface{}{
			"error": "engine not initialized",
		}))
		return
	}
	emitter := NewSessionEmitter(eb, sess.ID)
	_, err := eng.RunStream(sess.Ctx, sess.Messages, emitter, sess.ID)
	if err != nil {
		eb.Publish(sess.ID, events.NewEvent(events.EventError, sess.ID, map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}
	sm.CompleteSession(sess.ID)
}
