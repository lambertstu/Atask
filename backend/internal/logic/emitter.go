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

func RunAgent(em *engine.EngineManager, sm *session.SessionManager, eb *events.EventBus, sess *session.Session) {
	if em == nil {
		eb.Publish(sess.ID, events.NewEvent(events.EventError, sess.ID, map[string]interface{}{
			"error": "EngineManager not initialized",
		}))
		sm.CompleteSession(sess.ID)
		return
	}

	engCtx := em.GetOrCreate(sess.ProjectPath)
	if engCtx == nil || engCtx.Engine == nil {
		eb.Publish(sess.ID, events.NewEvent(events.EventError, sess.ID, map[string]interface{}{
			"error": "engine not initialized for project: " + sess.ProjectPath,
		}))
		return
	}

	engCtx.Engine.SetPermissionManager(sess.PermissionMgr)

	emitter := NewSessionEmitter(eb, sess.ID)
	updatedMessages, err := engCtx.Engine.RunStream(sess.Ctx, sess.Messages, emitter, sess.ID, sm)

	if len(updatedMessages) > 0 {
		sm.UpdateMessages(sess.ID, updatedMessages)
	}

	if err != nil {
		eb.Publish(sess.ID, events.NewEvent(events.EventError, sess.ID, map[string]interface{}{
			"error": err.Error(),
		}))
		sm.CompleteSession(sess.ID)
		return
	}

	sm.CompleteSession(sess.ID)
}
