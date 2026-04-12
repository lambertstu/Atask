# Atask Agent 开发指南

## 项目概述

Atask 是一个可视化看板界面的 AI Agent 框架，使用 Go 语言编写，后端位于 `backend/` 目录。

## 开发命令

### 后端

```bash
cd backend && make test
make test-coverage
make test-unit
make test-race
make test-security  # 安全模块
make test-utils     # 工具函数
make test-engine    # 引擎核心
make test-tools     # 工具系统
make test-systems   # 子系统
```

### 前端

```bash
cd ui/web && npm install
npm run dev         # 开发模式 (端口 3000)
npm run build       # 生产构建
npm run lint        # 代码检查
```

### 运行服务

```bash
# 终端模式
cd backend && go run ./cmd/agent

# Web Server 模式 (端口 8080)
cd backend && go run ./cmd/server

# 前端开发服务器
cd ui/web && npm run dev
```

## 架构要点

### 后端架构

- **双入口模式**: 
  - `backend/cmd/agent/` - 终端 REPL 模式
  - `backend/cmd/server/` - Web API Server (HTTP + WebSocket)
- **核心引擎**: `internal/engine/` - Agent 运行循环、上下文管理、恢复机制
- **会话管理**: `internal/session/` - 多会话管理、EventBus 事件推送、状态流转
- **API 层**: `internal/api/` - HTTP REST API + WebSocket 实时通信
- **工具注册**: `internal/tools/` - 工具接口和内置工具 (bash, read, write, edit)
- **子系统**: `internal/systems/` - memory, skills, tasks, subagent
- **权限控制**: `pkg/security/` - 模式切换 (plan/build)、路径白名单

### 前端架构

- **框架**: React 18 + TypeScript + Vite
- **UI 组件**: Ant Design 5 + Tailwind CSS
- **状态管理**: Zustand
- **实时通信**: WebSocket (支持自动重连)
- **核心组件**:
  - `Board` - 看板主界面 (5 列状态展示)
  - `Column` - 列组件 (Planning/Scheduled/In Progress/Human Review/Completed)
  - `SessionCard` - 会话卡片 (支持权限审批)
  - `Sidebar` - 工作区切换
  - `TopBar` - 搜索、新建会话、通知

### 会话状态流转

```
planning → scheduled → in_processing → completed
                    ↘ human_review (权限等待)
```

## 环境配置

### 后端

- 必须设置 `DASHSCOPE_API_KEY` 环境变量
- 配置从 `.env` 文件加载（向上递归查找）
- 默认模型：`glm-5`，API 端点：阿里云 DashScope
- Web Server 默认端口：`8080`

### 前端

- 开发服务器端口：`3000`
- API 代理：`/api` → `http://localhost:8080`
- WebSocket：`ws://localhost:8080/api/ws`

## 关键约定

- 模块名：`agent-base`（导入路径使用此名称）
- 工具接口：必须实现 `Name()`, `Description()`, `Execute()`, `Schema()` 方法
- 测试文件命名：`*_test.go`，使用 `stretchr/testify`
- JSON 文件被 gitignore（`*.json`, `*.jsonl`）
- 前端代码：TypeScript + ESLint 严格模式

## API 接口

### HTTP REST

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/sessions` | 获取所有会话 |
| POST | `/api/sessions` | 创建新会话 |
| POST | `/api/sessions/:id/messages` | 发送消息 |
| POST | `/api/sessions/:id/control` | 控制会话 (start/pause/cancel) |
| POST | `/api/sessions/:id/approve` | 批准权限请求 |
| POST | `/api/sessions/:id/reject` | 拒绝权限请求 |

### WebSocket 事件

**前端 → 后端**
- `subscribe` - 订阅会话事件
- `send_message` - 发送消息
- `approve` - 批准权限
- `reject` - 拒绝权限
- `control` - 控制会话

**后端 → 前端**
- `status_change` - 状态变化
- `output` - Agent 输出
- `tool_call` - 工具调用
- `permission_request` - 权限请求
- `error` - 错误

## 目录结构

```
Atask/
├── backend/
│   ├── cmd/
│   │   ├── agent/        # 终端入口
│   │   └── server/       # Web Server 入口
│   ├── internal/
│   │   ├── api/          # API 层 (HTTP + WebSocket)
│   │   ├── session/      # 会话管理
│   │   ├── engine/       # Agent 引擎
│   │   ├── tools/        # 工具系统
│   │   └── systems/      # 子系统
│   └── pkg/
│       ├── api/          # 共享类型
│       └── security/     # 权限管理
└── ui/web/
    ├── src/
    │   ├── components/   # React 组件
    │   ├── hooks/        # 自定义 Hooks
    │   ├── services/     # API + WebSocket
    │   ├── store/        # Zustand Store
    │   └── types/        # TypeScript 类型
    └── package.json
```

## 架构要点

- **入口点**: `backend/cmd/agent/main.go`
- **核心引擎**: `internal/engine/` - Agent 运行循环、上下文管理、恢复机制
- **工具注册**: `internal/tools/` - 工具接口和内置工具 (bash, read, write, edit)
- **子系统**: `internal/systems/` - memory, skills, tasks, subagent
- **权限控制**: `pkg/security/` - 模式切换 (plan/build)、路径白名单

## 环境配置

- 必须设置 `DASHSCOPE_API_KEY` 环境变量
- 配置从 `.env` 文件加载（向上递归查找）
- 默认模型: `glm-5`，API 端点: 阿里云 DashScope

## 关键约定

- 模块名: `agent-base`（导入路径使用此名称）
- 工具接口: 必须实现 `Name()`, `Description()`, `Execute()`, `Schema()` 方法
- 测试文件命名: `*_test.go`，使用 `stretchr/testify`
- JSON 文件被 gitignore（`*.json`, `*.jsonl`）

## 目录结构

```
backend/
├── cmd/agent/        # 入口点
├── internal/
│   ├── config/      # 配置加载
│   ├── engine/      # Agent 引擎核心
│   ├── llm/         # LLM 客户端封装
│   ├── systems/     # 子系统 (memory/skills/tasks/subagent)
│   └── tools/       # 工具注册和内置工具
├── pkg/
│   ├── events/      # Hook 系统
│   ├── security/    # 权限管理
│   └── utils/       # 工具函数
└── testutil/        # 测试辅助
```