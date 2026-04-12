import axios from 'axios'
import type { Session } from '../types'

const API_BASE = '/api'

export const api = {
  sessions: {
    list: async () => {
      const response = await axios.get<Session[]>(`${API_BASE}/sessions`)
      return response.data
    },
    
    create: async (name: string, workDir?: string) => {
      const response = await axios.post<Session>(`${API_BASE}/sessions`, { name, work_dir: workDir })
      return response.data
    },
    
    sendMessage: async (id: string, content: string) => {
      await axios.post(`${API_BASE}/sessions/${id}/messages`, { content })
    },
    
    control: async (id: string, action: 'start' | 'pause' | 'cancel') => {
      await axios.post(`${API_BASE}/sessions/${id}/control`, { action })
    },
    
    approve: async (id: string, requestId: string) => {
      await axios.post(`${API_BASE}/sessions/${id}/approve`, { request_id: requestId })
    },
    
    reject: async (id: string, requestId: string) => {
      await axios.post(`${API_BASE}/sessions/${id}/reject`, { request_id: requestId })
    },
  },
  
  health: async () => {
    const response = await axios.get<{ status: string }>('/health')
    return response.data
  },
}
