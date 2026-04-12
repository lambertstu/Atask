import React from 'react'
import { Typography, Badge } from 'antd'
import type { Session, SessionStatus } from '../../types'
import { SessionCard } from './SessionCard'

const { Title } = Typography

interface ColumnProps {
  title: string
  status: SessionStatus
  sessions: Session[]
  onApprove?: (sessionId: string, requestId: string) => void
  onReject?: (sessionId: string, requestId: string) => void
}

const STATUS_COLORS: Record<SessionStatus, string> = {
  planning: 'rgb(139, 92, 246)',
  scheduled: 'rgb(59, 130, 246)',
  in_processing: 'rgb(34, 197, 94)',
  human_review: 'rgb(249, 115, 22)',
  completed: 'rgb(156, 163, 175)',
}

export const Column: React.FC<ColumnProps> = ({ title, status, sessions, onApprove, onReject }) => {
  return (
    <div 
      className="flex-shrink-0 w-80 bg-gray-100 rounded-lg p-3"
      style={{ borderTop: `4px solid ${STATUS_COLORS[status]}` }}
    >
      <div className="mb-3 flex justify-between items-center">
        <Title level={5} className="!mb-0 !text-gray-700">{title}</Title>
        <Badge count={sessions.length} overflowCount={99} />
      </div>
      
      <div className="space-y-2 max-h-[calc(100vh-200px)] overflow-y-auto">
        {sessions.map(session => (
          <SessionCard 
            key={session.id} 
            session={session}
            onApprove={onApprove ? (requestId) => onApprove(session.id, requestId) : undefined}
            onReject={onReject ? (requestId) => onReject(session.id, requestId) : undefined}
          />
        ))}
        
        {sessions.length === 0 && (
          <div className="text-center text-gray-400 text-sm py-4">
            No sessions
          </div>
        )}
      </div>
    </div>
  )
}
