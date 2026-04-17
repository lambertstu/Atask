# AGENTS.md

## 项目概览

这是一个 Go 语言实现的 AI Agent 框架，提供可视化看板界面。项目采用 REPL（交互式命令行）和 REST API 两种运行模式，集成了 LLM、任务管理、内存系统、技能加载和子代理等子系统。

## 后端项目结构

```
backend/
├── agent.go                   # REST API 入口（go-zero 框架）
├── generate.go                # goctl 代码生成指令
├── cmd/
│   └── agent/main.go          # REPL 入口（交互式命令行）
├── api/
│   └── agent.api              # go-zero API 定义文件
├── etc/
│   └── agent-api.yaml         # go-zero 配置文件（goctl 生成）
├── internal/                  # 私有应用代码（goctl 标准结构）
│   ├── config/               # 配置加载（嵌入 rest.RestConf）
│   ├── handler/              # HTTP Handlers（goctl 生成，可编辑）
│   │   ├── routes.go         # 路由注册（自动生成，勿手动编辑）
│   │   └── *.go              # 各端点 handler（可编辑）
│   ├── logic/                # 业务逻辑层（goctl 骨架，实现业务）
│   │   └── *.go              # 各端点 logic（填充业务代码）
│   ├── svc/                  # 服务上下文（依赖注入）
│   │   └── servicecontext.go # ServiceContext 结构
│   ├── types/                # 类型定义（自动生成，勿手动编辑）
│   │   └── types.go          # Request/Response 结构
│   ├── engine/               # Agent 引擎核心
│   ├── llm/                  # LLM 客户端
│   ├── systems/              # 子系统
│   └── tools/                # 工具实现
├── pkg/                       # 公共包
│   ├── events/               # EventBus + Hook 管理
│   ├── security/             # 权限管理
│   └── utils/                # 工具函数
├── test/                      # 集成测试
├── testutil/                  # 测试工具
├── Makefile                  # 测试命令
├── go.mod                    # Go 模块定义
└── .env                      # 环境变量（需创建）
```

## 环境配置

### REST API 模式配置 (etc/agent-api.yaml)

REST API 模式使用 YAML 配置文件：

```yaml
Name: agent-api
Host: 0.0.0.0
Port: 8888
Model: glm-5
WorkDir: /path/to/your/project
ProjectRoot: /path/to/Atask
APIKey: your_api_key_here
BaseURL: "https://coding.dashscope.aliyuncs.com/v1"
ContextThreshold: 50000
BashTimeout: 120
```

### REPL 模式配置 (.env)

REPL 模式使用 `.env` 文件：

```bash
DASHSCOPE_API_KEY=your_api_key_here
```

**重要：** 
- REST API 模式：配置在 `etc/agent-api.yaml` 中
- REPL 模式：配置加载代码会从当前目录向上查找 `.env` 文件（最多 10 层目录）
- 默认使用阿里云通义千问 API（BaseURL: `https://coding.dashscope.aliyuncs.com/v1`）
- 默认模型：`glm-5`

## 开发命令

### 运行后端

```bash
cd backend

# REST API 模式（go-zero，推荐）
go run agent.go -f etc/agent-api.yaml

# REPL 模式（交互式命令行）
go run cmd/agent/main.go
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

1. **两种运行模式**: 
   - REPL 模式 (`cmd/agent/main.go`): 交互式命令行，支持 `/mode`, `/tasks` 等命令
   - REST API 模式 (`agent.go`): go-zero 框架，支持 Flutter 前端
2. **工作目录**: Agent 会在 `.memory/`、`.tasks/`、`.runtime-tasks/`、`.sessions/` 等目录中持久化数据
3. **项目根查找**: 通过查找 `.git` 目录确定项目根路径（向上遍历 20 层）
4. **技能目录**: 技能从 `<project-root>/skills/` 目录加载
5. **Git 忽略**: `.env`、`*.json`、`*.jsonl`、`*.log` 等文件已在 `.gitignore` 中配置
6. **Go 版本**: 需要 Go 1.24.0 或更高版本

## goctl 开发规范

### API 定义文件 (api/*.api)

定义 REST API 的请求/响应结构和路由：

```go
syntax = "v1"

type (
    CreateSessionRequest {
        ProjectPath string `json:"project_path"`
        Model       string `json:"model,optional"`
    }
    
    SessionResponse {
        ID          string `json:"id"`
        ProjectPath string `json:"project_path"`
        Model       string `json:"model"`
        State       string `json:"state"`
        CreatedAt   string `json:"created_at"`
    }
)

@server(
    prefix: /api
)
service agent-api {
    @handler CreateSession
    post /sessions (CreateSessionRequest) returns (SessionResponse)
}
```

### 代码生成命令

```bash
cd backend

# 从 api 文件生成代码（自动创建 handler/logic/svc/types）
goctl api go -api api/agent.api -dir .

# 验证 api 文件语法
goctl api validate -api api/agent.api

# 格式化 api 文件
goctl api format -api api/agent.api
```

### goctl 生成的目录结构

```
internal/
├── config/config.go          # 配置结构（需手动嵌入 rest.RestConf）
├── handler/
│   ├── routes.go             # 路由注册（自动生成，勿手动编辑）
│   └── createsessionhandler.go  # Handler（可编辑）
├── logic/
│   └── createsessionlogic.go    # 业务逻辑（填充实现）
├── svc/servicecontext.go     # 服务上下文（添加依赖注入）
└── types/types.go            # 类型定义（自动生成，勿手动编辑）
etc/
└── agent-api.yaml            # YAML 配置（自动生成）
agent.go                       # 入口文件（goctl 生成）
generate.go                    # 代码生成指令
```

### 开发流程

1. **修改 API 定义**: 编辑 `api/agent.api`
2. **重新生成代码**: `goctl api go -api api/agent.api -dir .`
3. **实现业务逻辑**: 编辑 `internal/logic/*.go`
4. **配置依赖注入**: 编辑 `internal/svc/servicecontext.go`
5. **运行服务**: `go run agent.go -f etc/agent-api.yaml`

### 文件编辑规则

| 文件 | 规则 |
|------|------|
| `internal/types/types.go` | 勿手动编辑，每次重新生成会覆盖 |
| `internal/handler/routes.go` | 勿手动编辑，每次重新生成会覆盖 |
| `internal/handler/*.handler.go` | 可编辑，但建议只在 logic 层实现业务 |
| `internal/logic/*.go` | 可编辑，填充业务逻辑 |
| `internal/svc/servicecontext.go` | 可编辑，添加服务依赖 |
| `internal/config/config.go` | 可编辑，嵌入 rest.RestConf |

### Config 结构规范

```go
type Config struct {
    rest.RestConf  // 必须嵌入（提供 Host/Port）
    // 自定义配置字段...
}
```

### ServiceContext 规范

```go
type ServiceContext struct {
    Config  config.Config
    // 依赖注入...
    SessionManager *session.SessionManager
    Engine         *engine.AgentEngine
    EventBus       *events.EventBus
}
```

### Logic 层规范

```go
func (l *CreateSessionLogic) CreateSession(req *types.CreateSessionRequest) (*types.SessionResponse, error) {
    // 从 svcCtx 获取依赖
    sm := l.svcCtx.SessionManager
    
    // 实现业务逻辑
    sess := sm.CreateSession(req.ProjectPath, req.Model)
    
    // 返回响应
    return &types.SessionResponse{
        ID:          sess.ID,
        ProjectPath: sess.ProjectPath,
        Model:       sess.Model,
        State:       string(sess.State),
        CreatedAt:   sess.CreatedAt.Format(time.RFC3339),
    }, nil
}
```

### SSE Handler 规范

goctl 生成的 SSE handler 使用 channel 模式：

```go
// Handler 层（自动生成）
func StreamSessionEventsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        client := make(chan *types.SessionEvent, 16)
        l := logic.NewStreamSessionEventsLogic(r.Context(), svcCtx)
        
        threading.GoSafeCtx(r.Context(), func() {
            defer close(client)
            l.StreamSessionEvents(client)  // Logic 向 channel 发送事件
        })
        
        // Handler 从 channel 读取并写入 HTTP Response
        for data := range client {
            fmt.Fprintf(w, "data: %s\n\n", json.Marshal(data))
            w.(http.Flusher).Flush()
        }
    }
}

// Logic 层（需实现）
func (l *StreamSessionEventsLogic) StreamSessionEvents(client chan<- *types.SessionEvent) error {
    eventCh, subscriberID := l.svcCtx.EventBus.Subscribe(sessionID)
    defer l.svcCtx.EventBus.Unsubscribe(sessionID, subscriberID)
    
    for {
        select {
        case event := <-eventCh:
            client <- convertEvent(event)
        case <-l.ctx.Done():
            return nil
        }
    }
}
```