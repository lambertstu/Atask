package events

import (
	"encoding/json"
	"fmt"
	"sync"
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
	id       string
	channel  chan Event
	doneChan chan struct{}
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
		id:       subID,
		channel:  make(chan Event, EventChannelBufferSize),
		doneChan: make(chan struct{}),
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

	close(sub.doneChan)
	close(sub.channel)
	delete(sessionSubs, subscriberID)

	if len(sessionSubs) == 0 {
		delete(b.subscribers, sessionID)
	}
}

func (b *EventBus) Publish(sessionID string, event Event) {
	b.mu.RLock()
	sessionSubs, ok := b.subscribers[sessionID]
	b.mu.RUnlock()

	if !ok {
		return
	}

	for _, sub := range sessionSubs {
		select {
		case sub.channel <- event:
		default:
		}
	}
}

func (b *EventBus) PublishGlobal(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, sessionSubs := range b.subscribers {
		for _, sub := range sessionSubs {
			select {
			case sub.channel <- event:
			default:
			}
		}
	}
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
		close(sub.doneChan)
		close(sub.channel)
	}
	delete(b.subscribers, sessionID)
}

func generateSubscriberID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}
