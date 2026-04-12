import React, { useState } from 'react'
import { Input, Button, Avatar, Dropdown, Badge, Space } from 'antd'
import { SearchOutlined, PlusOutlined, BellOutlined } from '@ant-design/icons'
import { useSessionStore } from '../../store'
import { api } from '../../services/api'
import { message } from 'antd'

const { Search } = Input

export const TopBar: React.FC = () => {
  const [searchValue, setSearchValue] = useState('')
  const [loading, setLoading] = useState(false)
  const addSession = useSessionStore(state => state.addSession)
  const wsConnected = useSessionStore(state => state.wsConnected)

  const handleNewSession = async () => {
    const name = prompt('Enter session name:')
    if (!name) return

    try {
      setLoading(true)
      const session = await api.sessions.create(name)
      addSession(session)
      message.success('Session created')
    } catch (error) {
      message.error('Failed to create session')
    } finally {
      setLoading(false)
    }
  }

  const userMenu = {
    items: [
      { key: 'profile', label: 'Profile' },
      { key: 'settings', label: 'Settings' },
      { type: 'divider' },
      { key: 'logout', label: 'Logout' },
    ],
  }

  return (
    <div className="h-16 bg-white border-b border-gray-200 px-4 flex items-center justify-between">
      <div className="flex-1 max-w-xl">
        <Search
          placeholder="Search agents or tasks..."
          allowClear
          value={searchValue}
          onChange={e => setSearchValue(e.target.value)}
          prefix={<SearchOutlined className="text-gray-400" />}
          className="w-full"
        />
      </div>

      <Space size="large" className="flex items-center">
        <div className="flex items-center gap-2">
          <div className={`w-2 h-2 rounded-full ${wsConnected ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-xs text-gray-500">
            {wsConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>

        <Badge count={3} size="small">
          <Button icon={<BellOutlined />} size="large" />
        </Badge>

        <Button 
          type="primary" 
          icon={<PlusOutlined />} 
          onClick={handleNewSession}
          loading={loading}
        >
          New Session
        </Button>

        <Dropdown menu={userMenu} placement="bottomRight">
          <Avatar 
            src="https://api.dicebear.com/7.x/miniavs/svg?seed=sarah" 
            className="cursor-pointer"
            size="large"
          />
        </Dropdown>
      </Space>
    </div>
  )
}
