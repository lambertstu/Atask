package session

import (
	"sync"
	"time"
)

type EventType string

const (
	EventStatusChange EventType = "status_change"
	EventOutput       EventType = "output"
	EventToolCall     EventType = "tool_call"
	EventError        EventType = "error"
	EventSubagent     EventType = "subagent"
	EventPermission   EventType = "permission_request"
)

type SessionEvent struct {
	SessionID string
	Type      EventType
	Timestamp int64
	Data      map[string]interface{}
}

type EventBus struct {
	subscribers map[string][]chan SessionEvent
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan SessionEvent),
	}
}

func (eb *EventBus) Subscribe(sessionID string) chan SessionEvent {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan SessionEvent, 100)
	eb.subscribers[sessionID] = append(eb.subscribers[sessionID], ch)
	return ch
}

func (eb *EventBus) Unsubscribe(sessionID string, ch chan SessionEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs := eb.subscribers[sessionID]
	for i, sub := range subs {
		if sub == ch {
			eb.subscribers[sessionID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

func (eb *EventBus) Publish(event SessionEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	eb.mu.RLock()
	subs := eb.subscribers[event.SessionID]
	eb.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (eb *EventBus) PublishAll(event SessionEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for sessionID, subs := range eb.subscribers {
		eventCopy := event
		eventCopy.SessionID = sessionID
		for _, ch := range subs {
			select {
			case ch <- eventCopy:
			default:
			}
		}
	}
}
