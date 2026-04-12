export type SessionStatus = 
  | 'planning' 
  | 'scheduled' 
  | 'in_processing' 
  | 'human_review' 
  | 'completed'

export interface Session {
  id: string
  name: string
  work_dir: string
  status: SessionStatus
  created_at: number
  last_active: number
  description?: string
}

export interface WSEvent {
  type: string
  session_id: string
  timestamp: number
  data: Record<string, any>
}

export interface PermissionRequest {
  request_id: string
  tool_name: string
  tool_input: Record<string, any>
  reason: string
}

export const STATUS_DISPLAY: Record<SessionStatus, string> = {
  planning: 'Planning',
  scheduled: 'Scheduled',
  in_processing: 'In Progress',
  human_review: 'Human Review',
  completed: 'Completed',
}

export const STATUS_COLORS: Record<SessionStatus, string> = {
  planning: 'purple',
  scheduled: 'blue',
  in_processing: 'green',
  human_review: 'orange',
  completed: 'gray',
}
