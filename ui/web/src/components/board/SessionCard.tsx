import React from 'react'
import { Card, Tag, Button, Progress, Typography, Space } from 'antd'
import type { Session } from '../../types'
import { STATUS_COLORS } from '../../types'

const { Text } = Typography

interface SessionCardProps {
  session: Session
  onApprove?: (requestId: string) => void
  onReject?: (requestId: string) => void
}

export const SessionCard: React.FC<SessionCardProps> = ({ session, onApprove, onReject }) => {
  const handleApprove = () => {
    if (session.permission_request && onApprove) {
      onApprove(session.permission_request.request_id)
    }
  }

  const handleReject = () => {
    if (session.permission_request && onReject) {
      onReject(session.permission_request.request_id)
    }
  }

  return (
    <Card 
      size="small" 
      className="mb-2 hover:shadow-md transition-shadow cursor-pointer"
      title={
        <div className="flex justify-between items-center">
          <Text strong className="truncate">{session.name}</Text>
          <Tag color={STATUS_COLORS[session.status]}>{session.status}</Tag>
        </div>
      }
    >
      <div className="text-xs text-gray-500 mb-2">
        <div className="truncate" title={session.work_dir}>
          📁 {session.work_dir.split('/').pop()}
        </div>
      </div>
      
      {session.description && (
        <div className="text-sm text-gray-600 mb-2 line-clamp-2">
          {session.description}
        </div>
      )}
      
      {session.status === 'in_processing' && (
        <Progress percent={session.progress || 35} size="small" strokeColor="#1890ff" />
      )}
      
      {session.status === 'human_review' && session.permission_request && (
        <div className="mt-2 p-2 bg-red-50 rounded border border-red-200">
          <div className="text-xs text-red-600 mb-2 truncate">
            ⚠️ {session.permission_request.reason}
          </div>
          <div className="flex gap-1">
            <Button 
              type="primary" 
              size="small" 
              onClick={handleApprove}
              className="flex-1"
            >
              ✓ Approve
            </Button>
            <Button 
              size="small" 
              danger 
              onClick={handleReject}
              className="flex-1"
            >
              ✕ Reject
            </Button>
          </div>
        </div>
      )}
      
      <div className="mt-2 text-xs text-gray-400">
        Last active: {new Date(session.last_active * 1000).toLocaleString()}
      </div>
    </Card>
  )
}
