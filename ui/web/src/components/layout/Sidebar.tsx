import React from 'react'
import { Menu, Typography } from 'antd'
import { useSessionStore } from '../../store'

const { Title } = Typography

export const Sidebar: React.FC = () => {
  const activeWorkspace = useSessionStore(state => state.activeWorkspace)
  const setActiveWorkspace = useSessionStore(state => state.setActiveWorkspace)

  const workspaces = [
    { key: 'q4-analysis', name: 'Q4 Market Analysis', color: 'purple' },
    { key: 'competitor', name: 'Competitor Tracking', color: 'blue' },
    { key: 'support', name: 'Customer Support', color: 'green' },
    { key: 'content', name: 'Content Strategy', color: 'orange' },
  ]

  return (
    <div className="w-64 bg-white border-r border-gray-200 flex flex-col h-full">
      <div className="p-4 border-b border-gray-200">
        <Title level={3} className="!mb-0 text-purple-600">
          <span role="img" aria-label="task">📊</span> Atask
        </Title>
      </div>
      
      <div className="p-4">
        <div className="text-xs text-gray-500 uppercase mb-2">Workspaces</div>
        <Menu
          mode="inline"
          selectedKeys={activeWorkspace ? [activeWorkspace] : []}
          className="border-none"
          items={workspaces.map(ws => ({
            key: ws.key,
            label: (
              <div className="flex items-center gap-2">
                <div className={`w-2 h-2 rounded-full bg-${ws.color}-500`} />
                <div>
                  <div className="text-sm">{ws.name}</div>
                  <div className="text-xs text-gray-400">Active</div>
                </div>
              </div>
            ),
            onClick: () => setActiveWorkspace(ws.key),
          }))}
        />
      </div>
    </div>
  )
}
