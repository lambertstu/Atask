import React, { useEffect } from 'react'
import { ConfigProvider } from 'antd'
import { Sidebar } from './components/layout/Sidebar'
import { TopBar } from './components/layout/TopBar'
import { Board } from './components/board/Board'
import { useSessionStore } from './store'
import { wsService } from './services/websocket'

const App: React.FC = () => {
  const setWsConnected = useSessionStore(state => state.setWsConnected)

  useEffect(() => {
    wsService.connect(`ws://${window.location.host}/api/ws`)
    
    const checkHealth = async () => {
      try {
        await fetch('/api/health')
        setWsConnected(true)
      } catch {
        setWsConnected(false)
      }
    }
    
    checkHealth()
    const interval = setInterval(checkHealth, 30000)
    
    return () => {
      clearInterval(interval)
      wsService.disconnect()
    }
  }, [setWsConnected])

  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#8b5cf6',
          borderRadius: 6,
        },
      }}
    >
      <div className="h-screen flex flex-col bg-gray-50">
        <TopBar />
        <div className="flex-1 flex overflow-hidden">
          <Sidebar />
          <Board />
        </div>
      </div>
    </ConfigProvider>
  )
}

export default App
