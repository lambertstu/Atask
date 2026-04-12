import React, { useEffect, useState } from 'react'
import { Column } from './Column'
import { useSessionStore } from '../../store'
import { useBoardEvents } from '../../hooks/useBoardEvents'
import { api } from '../../services/api'
import { wsService } from '../../services/websocket'
import type { SessionStatus } from '../../types'
import { message } from 'antd'

const COLUMNS: Array<{ key: SessionStatus; title: string; count?: number }> = [
  { key: 'planning', title: 'Planning (Backlog)' },
  { key: 'scheduled', title: 'Scheduled (Queued)' },
  { key: 'in_processing', title: 'In Progress' },
  { key: 'human_review', title: 'Human Review' },
  { key: 'completed', title: 'Completed' },
]

export const Board: React.FC = () => {
  const sessions = useSessionStore(state => state.sessions)
  const setSessions = useSessionStore(state => state.setSessions)
  const updateSession = useSessionStore(state => state.updateSession)
  const { subscribeToSessions, approvePermission, rejectPermission } = useBoardEvents()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadSessions()
    wsService.connect(`ws://${window.location.host}/api/ws`)
    
    return () => {
      wsService.disconnect()
    }
  }, [])

  useEffect(() => {
    if (sessions.length > 0) {
      subscribeToSessions(sessions.map(s => s.id))
    }
  }, [sessions])

  const loadSessions = async () => {
    try {
      setLoading(true)
      const data = await api.sessions.list()
      setSessions(data)
    } catch (error) {
      message.error('Failed to load sessions')
      console.error('Failed to load sessions:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleApprove = async (sessionId: string, requestId: string) => {
    try {
      await api.sessions.approve(sessionId, requestId)
      updateSession(sessionId, { permission_request: undefined })
      message.success('Permission approved')
    } catch (error) {
      message.error('Failed to approve permission')
    }
  }

  const handleReject = async (sessionId: string, requestId: string) => {
    try {
      await api.sessions.reject(sessionId, requestId)
      updateSession(sessionId, { permission_request: undefined })
      message.success('Permission rejected')
    } catch (error) {
      message.error('Failed to reject permission')
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">Loading sessions...</div>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-x-auto p-4">
      <div className="flex gap-4 h-full">
        {COLUMNS.map(column => {
          const columnSessions = sessions.filter(s => s.status === column.key)
          return (
            <Column
              key={column.key}
              title={column.title}
              status={column.key}
              sessions={columnSessions}
              onApprove={handleApprove}
              onReject={handleReject}
            />
          )
        })}
      </div>
    </div>
  )
}
