package sse

import (
	"fmt"
	"net/http"

	"agent-base/pkg/events"
)

type Handler struct {
	eventBus *events.EventBus
}

func NewHandler(eb *events.EventBus) *Handler {
	return &Handler{
		eventBus: eb,
	}
}

func (h *Handler) StreamSessionEvents(w http.ResponseWriter, r *http.Request) {
	sessionID := extractSessionID(r)
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	eventCh, subscriberID := h.eventBus.Subscribe(sessionID)
	defer h.eventBus.Unsubscribe(sessionID, subscriberID)

	fmt.Fprintf(w, "data: %s\n\n", formatConnectedEvent(sessionID))
	flusher.Flush()

	for {
		select {
		case event := <-eventCh:
			fmt.Fprintf(w, "data: %s\n\n", event.JSON())
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func extractSessionID(r *http.Request) string {
	path := r.URL.Path
	parts := splitPath(path)
	for i, part := range parts {
		if part == "sessions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return r.URL.Query().Get("session_id")
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

func formatConnectedEvent(sessionID string) string {
	return fmt.Sprintf(`{"type":"connected","session_id":"%s"}`, sessionID)
}
