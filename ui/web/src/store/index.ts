import { create } from 'zustand'
import type { Session, SessionStatus } from '../types'

interface SessionStore {
  sessions: Session[]
  activeWorkspace: string | null
  wsConnected: boolean
  
  setSessions: (sessions: Session[]) => void
  addSession: (session: Session) => void
  updateSession: (id: string, updates: Partial<Session>) => void
  moveSession: (id: string, status: SessionStatus) => void
  setActiveWorkspace: (workspace: string) => void
  setWsConnected: (connected: boolean) => void
}

export const useSessionStore = create<SessionStore>((set) => ({
  sessions: [],
  activeWorkspace: null,
  wsConnected: false,
  
  setSessions: (sessions) => set({ sessions }),
  addSession: (session) => set((state) => ({ 
    sessions: [...state.sessions, session] 
  })),
  updateSession: (id, updates) => set((state) => ({
    sessions: state.sessions.map(s => 
      s.id === id ? { ...s, ...updates } : s
    ),
  })),
  moveSession: (id, status) => set((state) => ({
    sessions: state.sessions.map(s =>
      s.id === id ? { ...s, status } : s
    ),
  })),
  setActiveWorkspace: (workspace) => set({ activeWorkspace: workspace }),
  setWsConnected: (connected) => set({ wsConnected: connected }),
}))
