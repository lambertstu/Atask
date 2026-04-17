package engine

import (
	"agent-base/internal/config"
	"agent-base/internal/llm"
	"agent-base/internal/systems/memory"
	"agent-base/internal/systems/skills"
	"agent-base/internal/systems/subagent"
	"agent-base/internal/systems/tasks"
	"agent-base/internal/tools"
	"agent-base/internal/tools/builtin"
	"agent-base/internal/tools/planning"
	"agent-base/pkg/events"
	"agent-base/pkg/security"
	"fmt"
	"path/filepath"
	"sync"
)

type EngineContext struct {
	Engine         *AgentEngine
	Registry       *tools.DefaultRegistry
	PermissionMgr  *security.PermissionManager
	HookMgr        *events.HookManager
	PromptBuilder  *SystemPromptBuilder
	ContextMgr     *ContextManagerImpl
	RecoveryMgr    *RecoveryManagerImpl
	CronScheduler  *tasks.CronScheduler
	MemoryMgr      *memory.MemoryManager
	TaskMgr        *tasks.TaskManager
	BackgroundMgr  *tasks.BackgroundManager
	SkillLoader    *skills.SkillLoader
	SubagentRunner *subagent.SubagentRunner
}

type EngineManager struct {
	contexts    map[string]*EngineContext
	mu          sync.RWMutex
	config      config.Config
	llmClient   *llm.DashScopeClient
	projectRoot string
}

func NewEngineManager(cfg config.Config, llmClient *llm.DashScopeClient) *EngineManager {
	return &EngineManager{
		contexts:    make(map[string]*EngineContext),
		config:      cfg,
		llmClient:   llmClient,
		projectRoot: cfg.ProjectRoot,
	}
}

func (em *EngineManager) GetOrCreate(workDir string) *EngineContext {
	em.mu.RLock()
	ctx, exists := em.contexts[workDir]
	em.mu.RUnlock()

	if exists {
		return ctx
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	if ctx, exists = em.contexts[workDir]; exists {
		return ctx
	}

	ctx = em.createEngineContext(workDir)
	em.contexts[workDir] = ctx
	return ctx
}

func (em *EngineManager) Get(workDir string) (*EngineContext, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()
	ctx, exists := em.contexts[workDir]
	return ctx, exists
}

func (em *EngineManager) createEngineContext(workDir string) *EngineContext {
	registry := tools.NewDefaultRegistry()
	registry.Register(builtin.NewBashTool(workDir, em.config.BashTimeout))
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

	skillsDir := filepath.Join(em.projectRoot, "skills")
	skillLoader := skills.NewSkillLoader(skillsDir)
	skillLoader.LoadAll()
	registry.Register(skills.NewLoadSkillTool(skillLoader))

	subagentRunner := subagent.NewSubagentRunner(em.llmClient, registry, workDir, em.config.Model)
	registry.Register(subagent.NewDelegateSubagentTool(subagentRunner))

	registry.Register(NewCompactTool())

	permissionMgr := security.NewPermissionManager("plan", workDir)
	hookMgr := events.NewHookManager(workDir)

	promptBuilder := NewSystemPromptBuilder(workDir, em.projectRoot, em.config.Model)
	contextMgr := NewContextManager(em.llmClient, em.config.Model, workDir, em.config.ContextThreshold)
	recoveryMgr := NewRecoveryManager(em.llmClient, em.config.Model, contextMgr, promptBuilder)

	agentEngine := NewAgentEngine(
		em.llmClient,
		registry,
		permissionMgr,
		hookMgr,
		promptBuilder,
		contextMgr,
		recoveryMgr,
		em.config.Model,
		em.config.ContextThreshold,
	)

	cronScheduler.Start()
	fmt.Printf("[EngineContext created for workDir: %s]\n", workDir)

	return &EngineContext{
		Engine:         agentEngine,
		Registry:       registry,
		PermissionMgr:  permissionMgr,
		HookMgr:        hookMgr,
		PromptBuilder:  promptBuilder,
		ContextMgr:     contextMgr,
		RecoveryMgr:    recoveryMgr,
		CronScheduler:  cronScheduler,
		MemoryMgr:      memoryMgr,
		TaskMgr:        taskMgr,
		BackgroundMgr:  backgroundMgr,
		SkillLoader:    skillLoader,
		SubagentRunner: subagentRunner,
	}
}

func (em *EngineManager) StopAll() {
	em.mu.Lock()
	defer em.mu.Unlock()

	for workDir, ctx := range em.contexts {
		if ctx.CronScheduler != nil {
			ctx.CronScheduler.Stop()
			fmt.Printf("[CronScheduler stopped for workDir: %s]\n", workDir)
		}
	}
}

func (em *EngineManager) ListWorkDirs() []string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var dirs []string
	for dir := range em.contexts {
		dirs = append(dirs, dir)
	}
	return dirs
}
