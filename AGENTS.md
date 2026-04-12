# Atask Agent 开发指南

## 项目概述

Atask 是一个可视化看板界面的 AI Agent 框架，使用 Go 语言编写，后端位于 `backend/` 目录。

## 开发命令

```bash
# 运行所有测试
cd backend && make test

# 生成覆盖率报告
make test-coverage

# 运行单元测试（跳过集成测试）
make test-unit

# 竞态检测
make test-race

# 按模块测试
make test-security  # 安全模块
make test-utils     # 工具函数
make test-engine    # 引擎核心
make test-tools     # 工具系统
make test-systems   # 子系统
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