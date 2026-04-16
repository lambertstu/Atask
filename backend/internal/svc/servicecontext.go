// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"agent-base/internal/config"
	"agent-base/internal/engine"
	"agent-base/internal/llm"
	"agent-base/internal/systems/memory"
	"agent-base/internal/systems/project"
	"agent-base/internal/systems/session"
	"agent-base/internal/systems/skills"
	"agent-base/internal/systems/subagent"
	"agent-base/internal/systems/tasks"
	"agent-base/internal/tools"
	"agent-base/internal/tools/builtin"
	"agent-base/internal/tools/planning"
	"agent-base/pkg/events"
	"agent-base/pkg/security"
	"path/filepath"
)

type ServiceContext struct {
	Config         config.Config
	SessionManager *session.SessionManager
	Engine         *engine.AgentEngine
	EventBus       *events.EventBus
	ProjectManager *project.ProjectManager
}

func NewServiceContext(cfg config.Config) *ServiceContext {
	llmClient := llm.NewClient(&cfg)

	registry, cronScheduler := registryTool(cfg.WorkDir, cfg.Model, llmClient)

	cronScheduler.Start()
	defer cronScheduler.Stop()

	permissionMgr := security.NewPermissionManager("plan", cfg.WorkDir)
	hookMgr := events.NewHookManager(cfg.WorkDir)

	promptBuilder := engine.NewSystemPromptBuilder(cfg.WorkDir, cfg.ProjectRoot, cfg.Model)
	contextMgr := engine.NewContextManager(llmClient, cfg.Model, cfg.WorkDir, cfg.ContextThreshold)
	recoveryMgr := engine.NewRecoveryManager(llmClient, cfg.Model, contextMgr, promptBuilder)

	eventBus := events.NewEventBus()

	sessionManager := session.NewSessionManager(cfg.ProjectRoot, eventBus, cfg.WorkDir)

	projectManager := project.NewProjectManager(cfg.ProjectRoot)

	agentEngine := engine.NewAgentEngine(
		llmClient,
		registry,
		permissionMgr,
		hookMgr,
		promptBuilder,
		contextMgr,
		recoveryMgr,
		cfg.Model,
		cfg.ContextThreshold,
	)

	return &ServiceContext{
		Config:         cfg,
		SessionManager: sessionManager,
		Engine:         agentEngine,
		EventBus:       eventBus,
		ProjectManager: projectManager,
	}
}

func registryTool(workDir, defaultModel string, llmClient *llm.DashScopeClient) (*tools.DefaultRegistry, *tasks.CronScheduler) {
	// registry里面的工作目录应该项目创建的时候初始化
	registry := tools.NewDefaultRegistry()
	registry.Register(builtin.NewBashTool(workDir, 120))
	registry.Register(builtin.NewReadFileTool(workDir))
	registry.Register(builtin.NewSearchFilesTool(workDir))
	registry.Register(builtin.NewGrepCodeTool(workDir))
	registry.Register(builtin.NewWriteFileTool(workDir))
	registry.Register(builtin.NewEditFileTool(workDir))
	registry.Register(builtin.NewWebFetchTool())
	registry.Register(planning.NewTodoTool())

	memoryMgr := memory.NewMemoryManager(filepath.Join(workDir, ".memory"))
	memoryMgr.LoadAll()
	registry.Register(memory.NewSaveMemoryTool(memoryMgr))

	tasksDir := filepath.Join(workDir, ".tasks")
	taskMgr := tasks.NewTaskManager(tasksDir)
	backgroundMgr := tasks.NewBackgroundManager(filepath.Join(workDir, ".runtime-tasks"))
	cronScheduler := tasks.NewCronScheduler()

	registry.Register(tasks.NewTaskCreateTool(taskMgr))
	registry.Register(tasks.NewTaskUpdateTool(taskMgr))
	registry.Register(tasks.NewTaskListTool(taskMgr))
	registry.Register(tasks.NewTaskGetTool(taskMgr))
	registry.Register(tasks.NewBackgroundRunTool(backgroundMgr))
	registry.Register(tasks.NewCheckBackgroundTool(backgroundMgr))
	registry.Register(tasks.NewCronCreateTool(cronScheduler))
	registry.Register(tasks.NewCronDeleteTool(cronScheduler))
	registry.Register(tasks.NewCronListTool(cronScheduler))

	// 注册 Skills 工具
	skillLoader := skills.NewSkillLoader(filepath.Join(workDir, "skills"))
	skillLoader.LoadAll()
	registry.Register(skills.NewLoadSkillTool(skillLoader))

	// 注册 Subagent 工具
	subagentRunner := subagent.NewSubagentRunner(llmClient, registry, workDir, defaultModel)
	registry.Register(subagent.NewDelegateSubagentTool(subagentRunner))

	// 注册 Compact 工具
	registry.Register(engine.NewCompactTool())
	return registry, cronScheduler
}
