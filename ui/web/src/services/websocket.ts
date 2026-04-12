import type { WSEvent } from '../types'

export class WebSocketService {
  private ws: WebSocket | null = null
  private eventHandlers: Map<string, Set<(event: WSEvent) => void>> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 3000

  connect(url: string) {
    try {
      this.ws = new WebSocket(url)
      
      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.reconnectAttempts = 0
      }
      
      this.ws.onmessage = (event) => {
        try {
          const wsEvent: WSEvent = JSON.parse(event.data)
          this.handleEvent(wsEvent)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }
      
      this.ws.onclose = () => {
        console.log('WebSocket closed')
        this.attemptReconnect(url)
      }
      
      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    } catch (err) {
      console.error('Failed to connect WebSocket:', err)
    }
  }
  
  private attemptReconnect(url: string) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`)
      setTimeout(() => this.connect(url), this.reconnectDelay)
    } else {
      console.error('Max reconnection attempts reached')
    }
  }
  
  subscribe(sessionIds: string[]) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'subscribe',
        session_ids: sessionIds,
      }))
    }
  }
  
  sendMessage(sessionId: string, content: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'send_message',
        session_id: sessionId,
        content,
      }))
    }
  }
  
  approvePermission(sessionId: string, requestId: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'approve',
        session_id: sessionId,
        request_id: requestId,
      }))
    }
  }
  
  rejectPermission(sessionId: string, requestId: string) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'reject',
        session_id: sessionId,
        request_id: requestId,
      }))
    }
  }
  
  controlSession(sessionId: string, action: 'start' | 'pause' | 'cancel') {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        type: 'control',
        session_id: sessionId,
        content: action,
      }))
    }
  }
  
  on(eventType: string, handler: (event: WSEvent) => void) {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, new Set())
    }
    this.eventHandlers.get(eventType)!.add(handler)
  }
  
  off(eventType: string, handler: (event: WSEvent) => void) {
    this.eventHandlers.get(eventType)?.delete(handler)
  }
  
  private handleEvent(event: WSEvent) {
    this.eventHandlers.get(event.type)?.forEach(handler => {
      try {
        handler(event)
      } catch (err) {
        console.error('Event handler error:', err)
      }
    })
  }
  
  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }
}

export const wsService = new WebSocketService()
