package events

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const EventChannelBufferSize = 100

type EventType string

const (
	EventStateChange      EventType = "state_change"
	EventThinking         EventType = "thinking"
	EventToolStart        EventType = "tool_start"
	EventToolEnd          EventType = "tool_end"
	EventAssistantMessage EventType = "assistant_message"
	EventBlocked          EventType = "blocked"
	EventCompleted        EventType = "completed"
	EventError            EventType = "error"
	EventRetry            EventType = "retry"
)

type Event struct {
	Type      EventType              `json:"type"`
	SessionID string                 `json:"session_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func (e Event) JSON() string {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"type":"%s","error":"marshal_error"}`, e.Type)
	}
	return string(data)
}

func NewEvent(eventType EventType, sessionID string, data map[string]interface{}) Event {
	return Event{
		Type:      eventType,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data:      data,
	}
}

type subscriber struct {
	id      string
	channel chan Event
}

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]*subscriber
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string]map[string]*subscriber),
	}
}

func (b *EventBus) Subscribe(sessionID string) (<-chan Event, string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers[sessionID] == nil {
		b.subscribers[sessionID] = make(map[string]*subscriber)
	}

	subID := generateSubscriberID()
	sub := &subscriber{
		id:      subID,
		channel: make(chan Event, EventChannelBufferSize),
	}
	b.subscribers[sessionID][subID] = sub

	return sub.channel, subID
}

func (b *EventBus) Unsubscribe(sessionID string, subscriberID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sessionSubs, ok := b.subscribers[sessionID]
	if !ok {
		return
	}

	sub, ok := sessionSubs[subscriberID]
	if !ok {
		return
	}

	close(sub.channel)
	delete(sessionSubs, subscriberID)

	if len(sessionSubs) == 0 {
		delete(b.subscribers, sessionID)
	}
}

func (b *EventBus) Publish(sessionID string, event Event) {
	b.mu.RLock()
	sessionSubs, ok := b.subscribers[sessionID]
	if !ok {
		b.mu.RUnlock()
		return
	}

	for _, sub := range sessionSubs {
		select {
		case sub.channel <- event:
		default:
		}
	}
	b.mu.RUnlock()
}

func (b *EventBus) PublishGlobal(event Event) {
	b.mu.RLock()
	for _, sessionSubs := range b.subscribers {
		for _, sub := range sessionSubs {
			select {
			case sub.channel <- event:
			default:
			}
		}
	}
	b.mu.RUnlock()
}

func (b *EventBus) HasSubscribers(sessionID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	sessionSubs, ok := b.subscribers[sessionID]
	if !ok {
		return false
	}
	return len(sessionSubs) > 0
}

func (b *EventBus) SubscriberCount(sessionID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	sessionSubs, ok := b.subscribers[sessionID]
	if !ok {
		return 0
	}
	return len(sessionSubs)
}

func (b *EventBus) CloseSession(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sessionSubs, ok := b.subscribers[sessionID]
	if !ok {
		return
	}

	for _, sub := range sessionSubs {
		close(sub.channel)
	}
	delete(b.subscribers, sessionID)
}

var subscriberCounter uint64

func generateSubscriberID() string {
	id := atomic.AddUint64(&subscriberCounter, 1)
	return fmt.Sprintf("sub_%d", id)
}
