# AGENTS.md

## 项目概览

这是一个 Go 语言实现的 AI Agent 框架，提供可视化看板界面。项目采用 REPL（交互式命令行）架构，集成了 LLM、任务管理、内存系统、技能加载和子代理等子系统。

## 后端项目结构

```
backend/
├── cmd/
│   ├── agent/main.go          # REPL 入口（交互式命令行）
│   ├── server/main.go         # WebSocket 服务器入口
│   └── server_rest/main.go    # REST+SSE 服务器入口（go-zero）
├── api/
│   └ agent.api                # go-zero API 定义文件
├── internal/                  # 私有应用代码
│   ├── config/               # 配置加载
│   ├── engine/               # Agent 引擎核心（主循环、上下文、提示、恢复）
│   │   ├── loop.go           # Run() 方法（REPL 用）
│   │   └── loop_stream.go    # RunStream() 方法（Server 用）
│   ├── llm/                  # LLM 客户端
│   ├── systems/              # 子系统
│   │   ├── memory/           # 持久化记忆（.memory/ 目录）
│   │   ├── project/          # 项目管理（多项目支持）
│   │   ├── session/          # 会话状态管理（支持事件发布）
│   │   ├── skills/           # 技能加载器（skills/ 目录）
│   │   ├── subagent/         # 子代理委托
│   │   └── tasks/            # 任务管理（.tasks/、.runtime-tasks/）
│   └── tools/                # 工具实现
│       ├── builtin/          # 内置工具
│       ├── planning/         # 规划工具（todo）
│       └── registry.go       # 工具注册中心
├── pkg/                       # 公共包
│   ├── api/
│   │   ├── rest/             # REST API handlers
│   │   ├── sse/              # SSE handlers
│   │   └── websocket/        # WebSocket API 层（兼容旧版）
│   ├── events/               # EventBus + Hook 管理
│   ├── security/             # 权限管理（支持会话级阻塞）
│   └── utils/                # 工具函数
├── test/                      # 集成测试
├── testutil/                  # 测试工具
├── Makefile                  # 测试命令
├── go.mod                    # Go 模块定义
└── .env                      # 环境变量（需创建）
```

## 环境配置

运行后端前需要创建 `backend/.env` 文件并设置：

```bash
DASHSCOPE_API_KEY=your_api_key_here
```

**重要：** 
- 配置加载代码会从当前目录向上查找 `.env` 文件（最多 10 层目录）
- 默认使用阿里云通义千问 API（BaseURL: `https://coding.dashscope.aliyuncs.com/v1`）
- 默认模型：`glm-5`

## 开发命令

### 运行后端

```bash
cd backend

# REPL 模式（交互式命令行）
go run cmd/agent/main.go

# WebSocket 服务器模式（旧版）
go run cmd/server/main.go

# REST+SSE 服务器模式（新版，推荐用于 Flutter 前端）
go run cmd/server_rest/main.go [--port=8080]
```

### REST+SSE API 端点

```
POST /api/sessions          # 创建会话
GET  /api/sessions          # 列出会话
GET  /api/sessions/:id      # 获取会话详情
POST /api/sessions/:id/input    # 提交输入
POST /api/sessions/:id/approve  # 批准计划
POST /api/sessions/:id/unblock  # 解除阻塞
GET  /api/sessions/:id/events   # SSE 事件流
```

### 测试

```bash
cd backend

# 运行所有测试
make test

# 运行单元测试（跳过集成测试）
make test-unit

# 生成覆盖率报告
make test-coverage

# 运行特定模块测试
make test-security    # pkg/security 测试
make test-utils       # pkg/utils 测试
make test-engine      # internal/engine 测试
make test-tools       # internal/tools 测试
make test-systems     # internal/systems 测试

# 检测竞态条件
make test-race
```

### 其他命令

```bash
# 清理覆盖率文件
make clean-coverage
```

## 架构要点

### 模块名称
- Go 模块名：`agent-base`（见 `go.mod`）
- 导入路径：`agent-base/internal/...` 和 `agent-base/pkg/...`

### 核心组件
1. **Agent Engine** (`internal/engine/`): 主循环、上下文管理、提示构建、恢复机制
2. **Tools Registry**: 注册内置工具（bash, read, write, edit, search, grep, webfetch, planning 等）
3. **Systems**: 
   - Memory: 持久化记忆（`.memory/` 目录）
   - Project: 多项目管理（`.projects/` 目录）
   - Session: 会话状态管理（`.sessions/` 目录，WebSocket 用）
   - Tasks: 任务管理（`.tasks/` 目录）
   - Skills: 技能加载器（从 `skills/` 目录加载）
   - Subagent: 子代理委托
4. **Security**: 权限管理（plan 模式）
5. **Events**: Hook 系统

### REPL 命令
Agent 启动后支持以下命令：
- `/mode <plan|build>` - 切换运行模式
- `/tasks` - 查看任务
- `/cron` - 查看定时任务
- `/memories` - 查看记忆
- `/prompt` - 查看当前提示
- `/compact` - 压缩上下文

## 测试约定

- 使用 `testify` 断言库
- `testutil/` 提供测试辅助工具：
  - `mock_llm.go`: Mock LLM 客户端
  - `fixtures.go`: 测试固件
  - `tempdir.go`: 临时目录管理
- 集成测试放在 `test/` 目录
- 使用 `-short` 标志区分单元测试和集成测试

## 依赖说明

- `github.com/sashabaranov/go-openai`: OpenAI 客户端（用于 LLM 调用）
- `github.com/joho/godotenv`: .env 文件加载
- `github.com/stretchr/testify`: 测试框架
- `github.com/JohannesKaufmann/html-to-markdown`: HTML 转 Markdown
- `github.com/gorilla/websocket`: WebSocket 服务器（可视化看板用）
- `github.com/zeromicro/go-zero`: REST+SSE 服务器框架

## 重要注意事项

1. **三种运行模式**: 
   - REPL 模式 (`cmd/agent/main.go`): 交互式命令行，支持 `/mode`, `/tasks` 等命令
   - WebSocket 模式 (`cmd/server/main.go`): 服务器监听 `:8080`，端点 `/ws`
   - REST+SSE 模式 (`cmd/server_rest/main.go`): go-zero 框架，支持 Flutter 前端
2. **工作目录**: Agent 会在 `.memory/`、`.tasks/`、`.runtime-tasks/`、`.sessions/` 等目录中持久化数据
3. **项目根查找**: 通过查找 `.git` 目录确定项目根路径（向上遍历 20 层）
4. **技能目录**: 技能从 `<project-root>/skills/` 目录加载
5. **Git 忽略**: `.env`、`*.json`、`*.jsonl`、`*.log` 等文件已在 `.gitignore` 中配置
6. **Go 版本**: 需要 Go 1.24.0 或更高版本