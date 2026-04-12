import { useEffect, useCallback } from 'react'
import { wsService } from '../services/websocket'
import { useSessionStore } from '../store'
import type { WSEvent, SessionStatus } from '../types'

export function useBoardEvents() {
  const updateSession = useSessionStore(state => state.updateSession)
  const moveSession = useSessionStore(state => state.moveSession)
  const addSession = useSessionStore(state => state.addSession)
  const setWsConnected = useSessionStore(state => state.setWsConnected)

  const handleStatusChange = useCallback((event: WSEvent) => {
    const { from, to } = event.data
    if (to === 'completed') {
      moveSession(event.session_id, to as SessionStatus)
    } else {
      updateSession(event.session_id, { status: to as SessionStatus })
    }
    console.log(`Session ${event.session_id}: ${from} -> ${to}`)
  }, [updateSession, moveSession])

  const handleOutput = useCallback((event: WSEvent) => {
    const { content } = event.data
    console.log(`Session ${event.session_id} output:`, content)
  }, [])

  const handleToolCall = useCallback((event: WSEvent) => {
    const { tool_name } = event.data
    console.log(`Session ${event.session_id} calling tool:`, tool_name)
  }, [])

  const handlePermission = useCallback((event: WSEvent) => {
    const { request_id, tool_name, reason } = event.data
    console.log(`Session ${event.session_id} permission request:`, { request_id, tool_name, reason })
    updateSession(event.session_id, { 
      permission_request: { request_id, tool_name, tool_input: event.data.tool_input, reason }
    })
  }, [updateSession])

  const handleError = useCallback((event: WSEvent) => {
    const { error } = event.data
    console.error(`Session ${event.session_id} error:`, error)
  }, [])

  useEffect(() => {
    wsService.on('status_change', handleStatusChange)
    wsService.on('output', handleOutput)
    wsService.on('tool_call', handleToolCall)
    wsService.on('permission_request', handlePermission)
    wsService.on('error', handleError)

    wsService.on('status_change', () => {})

    return () => {
      wsService.off('status_change', handleStatusChange)
      wsService.off('output', handleOutput)
      wsService.off('tool_call', handleToolCall)
      wsService.off('permission_request', handlePermission)
      wsService.off('error', handleError)
    }
  }, [handleStatusChange, handleOutput, handleToolCall, handlePermission, handleError])

  const subscribeToSessions = (sessionIds: string[]) => {
    wsService.subscribe(sessionIds)
  }

  const sendMessage = (sessionId: string, content: string) => {
    wsService.sendMessage(sessionId, content)
  }

  const approvePermission = (sessionId: string, requestId: string) => {
    wsService.approvePermission(sessionId, requestId)
  }

  const rejectPermission = (sessionId: string, requestId: string) => {
    wsService.rejectPermission(sessionId, requestId)
  }

  const controlSession = (sessionId: string, action: 'start' | 'pause' | 'cancel') => {
    wsService.controlSession(sessionId, action)
  }

  return {
    subscribeToSessions,
    sendMessage,
    approvePermission,
    rejectPermission,
    controlSession,
  }
}
