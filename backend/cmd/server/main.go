package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	"agent-base/pkg/api/websocket"
	"agent-base/pkg/events"
	"agent-base/pkg/security"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	llmClient := llm.NewClient(cfg)

	registry := tools.NewDefaultRegistry()
	registry.Register(builtin.NewBashTool(cfg.WorkDir, cfg.BashTimeout))
	registry.Register(builtin.NewReadFileTool(cfg.WorkDir))
	registry.Register(builtin.NewSearchFilesTool(cfg.WorkDir))
	registry.Register(builtin.NewGrepCodeTool(cfg.WorkDir))
	registry.Register(builtin.NewWriteFileTool(cfg.WorkDir))
	registry.Register(builtin.NewEditFileTool(cfg.WorkDir))
	registry.Register(builtin.NewWebFetchTool())
	registry.Register(planning.NewTodoTool())

	memoryMgr := memory.NewMemoryManager(filepath.Join(cfg.WorkDir, ".memory"))
	memoryMgr.LoadAll()
	registry.Register(memory.NewSaveMemoryTool(memoryMgr))

	tasksDir := filepath.Join(cfg.WorkDir, ".tasks")
	taskMgr := tasks.NewTaskManager(tasksDir)
	backgroundMgr := tasks.NewBackgroundManager(filepath.Join(cfg.WorkDir, ".runtime-tasks"))
	cronScheduler := tasks.NewCronScheduler()

	cronScheduler.Start()
	defer cronScheduler.Stop()

	registry.Register(tasks.NewTaskCreateTool(taskMgr))
	registry.Register(tasks.NewTaskUpdateTool(taskMgr))
	registry.Register(tasks.NewTaskListTool(taskMgr))
	registry.Register(tasks.NewTaskGetTool(taskMgr))
	registry.Register(tasks.NewBackgroundRunTool(backgroundMgr))
	registry.Register(tasks.NewCheckBackgroundTool(backgroundMgr))
	registry.Register(tasks.NewCronCreateTool(cronScheduler))
	registry.Register(tasks.NewCronDeleteTool(cronScheduler))
	registry.Register(tasks.NewCronListTool(cronScheduler))

	skillLoader := skills.NewSkillLoader(filepath.Join(cfg.ProjectRoot, "skills"))
	skillLoader.LoadAll()
	registry.Register(skills.NewLoadSkillTool(skillLoader))

	subagentRunner := subagent.NewSubagentRunner(llmClient, registry, cfg.WorkDir, cfg.Model)
	registry.Register(subagent.NewDelegateSubagentTool(subagentRunner))

	registry.Register(engine.NewCompactTool())

	permissionMgr := security.NewPermissionManager("plan", cfg.WorkDir)
	hookMgr := events.NewHookManager(cfg.WorkDir)

	promptBuilder := engine.NewSystemPromptBuilder(cfg.WorkDir, cfg.ProjectRoot, cfg.Model)
	contextMgr := engine.NewContextManager(llmClient, cfg.Model, cfg.WorkDir, cfg.ContextThreshold)
	recoveryMgr := engine.NewRecoveryManager(llmClient, cfg.Model, contextMgr, promptBuilder)

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

	projectManager := project.NewProjectManager(cfg.ProjectRoot)
	sessionManager := session.NewSessionManager(cfg.ProjectRoot)

	wsHandler := websocket.NewHandler(projectManager, agentEngine, permissionMgr)
	wsHandler.SetSessionManager(sessionManager)

	http.HandleFunc("/ws", wsHandler.HandleWS)

	addr := ":8080"
	log.Printf("WebSocket server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
