package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	assert.NotNil(t, bus)
	assert.NotNil(t, bus.subscribers)
}

func TestSubscribe(t *testing.T) {
	bus := NewEventBus()
	sessionID := "test-session-1"

	ch, subID := bus.Subscribe(sessionID)
	assert.NotNil(t, ch)
	assert.NotEmpty(t, subID)
	assert.True(t, bus.HasSubscribers(sessionID))
	assert.Equal(t, 1, bus.SubscriberCount(sessionID))
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	sessionID := "test-session-1"

	_, subID1 := bus.Subscribe(sessionID)
	_, subID2 := bus.Subscribe(sessionID)

	assert.NotEqual(t, subID1, subID2)
	assert.Equal(t, 2, bus.SubscriberCount(sessionID))

	bus.Unsubscribe(sessionID, subID1)
	assert.Equal(t, 1, bus.SubscriberCount(sessionID))

	bus.Unsubscribe(sessionID, subID2)
	assert.Equal(t, 0, bus.SubscriberCount(sessionID))
	assert.False(t, bus.HasSubscribers(sessionID))
}

func TestPublish(t *testing.T) {
	bus := NewEventBus()
	sessionID := "test-session-1"

	ch, subID := bus.Subscribe(sessionID)

	event := NewEvent(EventStateChange, sessionID, map[string]interface{}{
		"new_state": "processing",
	})

	bus.Publish(sessionID, event)

	select {
	case received := <-ch:
		assert.Equal(t, EventStateChange, received.Type)
		assert.Equal(t, sessionID, received.SessionID)
		assert.NotNil(t, received.Data)
		assert.Equal(t, "processing", received.Data["new_state"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive event but timed out")
	}

	bus.Unsubscribe(sessionID, subID)
}

func TestPublishToMultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	sessionID := "test-session-1"

	ch1, subID1 := bus.Subscribe(sessionID)
	ch2, subID2 := bus.Subscribe(sessionID)

	event := NewEvent(EventAssistantMessage, sessionID, map[string]interface{}{
		"content": "test message",
	})

	bus.Publish(sessionID, event)

	var wg sync.WaitGroup
	wg.Add(2)

	var received1, received2 Event

	go func() {
		defer wg.Done()
		received1 = <-ch1
	}()

	go func() {
		defer wg.Done()
		received2 = <-ch2
	}()

	wg.Wait()

	assert.Equal(t, event.Type, received1.Type)
	assert.Equal(t, event.Type, received2.Type)
	assert.Equal(t, "test message", received1.Data["content"])
	assert.Equal(t, "test message", received2.Data["content"])

	bus.Unsubscribe(sessionID, subID1)
	bus.Unsubscribe(sessionID, subID2)
}

func TestPublishNoSubscribers(t *testing.T) {
	bus := NewEventBus()
	sessionID := "non-existent-session"

	event := NewEvent(EventStateChange, sessionID, nil)

	bus.Publish(sessionID, event)
}

func TestUnsubscribeNonExistent(t *testing.T) {
	bus := NewEventBus()

	bus.Unsubscribe("non-existent-session", "non-existent-subscriber")

	bus.Subscribe("test-session")
	bus.Unsubscribe("test-session", "non-existent-subscriber")

	assert.Equal(t, 1, bus.SubscriberCount("test-session"))
}

func TestCloseSession(t *testing.T) {
	bus := NewEventBus()
	sessionID := "test-session-1"

	_, _ = bus.Subscribe(sessionID)
	_, _ = bus.Subscribe(sessionID)

	bus.CloseSession(sessionID)

	assert.False(t, bus.HasSubscribers(sessionID))
	assert.Equal(t, 0, bus.SubscriberCount(sessionID))
}

func TestEventJSON(t *testing.T) {
	event := NewEvent(EventStateChange, "session-1", map[string]interface{}{
		"new_state": "processing",
	})

	jsonStr := event.JSON()
	assert.Contains(t, jsonStr, "state_change")
	assert.Contains(t, jsonStr, "session-1")
	assert.Contains(t, jsonStr, "processing")
}

func TestPublishGlobal(t *testing.T) {
	bus := NewEventBus()

	ch1, subID1 := bus.Subscribe("session-1")
	ch2, subID2 := bus.Subscribe("session-2")

	event := NewEvent(EventError, "global", map[string]interface{}{
		"error": "global error",
	})

	bus.PublishGlobal(event)

	select {
	case received := <-ch1:
		assert.Equal(t, EventError, received.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive global event on session-1 channel")
	}

	select {
	case received := <-ch2:
		assert.Equal(t, EventError, received.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected to receive global event on session-2 channel")
	}

	bus.Unsubscribe("session-1", subID1)
	bus.Unsubscribe("session-2", subID2)
}

func TestBufferedChannelNotBlocking(t *testing.T) {
	bus := NewEventBus()
	sessionID := "test-session"

	ch, subID := bus.Subscribe(sessionID)

	for i := 0; i < EventChannelBufferSize+10; i++ {
		event := NewEvent(EventThinking, sessionID, map[string]interface{}{
			"iteration": i,
		})
		bus.Publish(sessionID, event)
	}

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			break
		}
		if count >= EventChannelBufferSize {
			break
		}
	}

	assert.Equal(t, EventChannelBufferSize, count)

	bus.Unsubscribe(sessionID, subID)
}
