package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"agent-base/internal/config"
	"agent-base/internal/engine"
	"agent-base/internal/llm"
	"agent-base/internal/systems/memory"
	"agent-base/internal/systems/session"
	"agent-base/internal/systems/skills"
	"agent-base/internal/systems/tasks"
	"agent-base/internal/tools"
	"agent-base/internal/tools/builtin"
	"agent-base/internal/tools/planning"
	apirest "agent-base/pkg/api/rest"
	sse "agent-base/pkg/api/sse"
	"agent-base/pkg/events"

	gorest "github.com/zeromicro/go-zero/rest"
)

func main() {
	port := flag.Int("port", 8080, "server port")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	llmClient := llm.NewClient(cfg)

	registry := tools.NewDefaultRegistry()
	registry.Register(builtin.NewBashTool(cfg.WorkDir, cfg.BashTimeout))
	registry.Register(builtin.NewReadFileTool(cfg.WorkDir))
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

	registry.Register(engine.NewCompactTool())

	hookMgr := events.NewHookManager(cfg.WorkDir)

	promptBuilder := engine.NewSystemPromptBuilder(cfg.WorkDir, cfg.ProjectRoot, cfg.Model)
	contextMgr := engine.NewContextManager(llmClient, cfg.Model, cfg.WorkDir, cfg.ContextThreshold)
	recoveryMgr := engine.NewRecoveryManager(llmClient, cfg.Model, contextMgr, promptBuilder)

	eventBus := events.NewEventBus()

	sessionManager := session.NewSessionManager(cfg.ProjectRoot, eventBus, cfg.WorkDir)

	agentEngine := engine.NewAgentEngine(
		llmClient,
		registry,
		nil,
		hookMgr,
		promptBuilder,
		contextMgr,
		recoveryMgr,
		cfg.Model,
		cfg.ContextThreshold,
	)

	restHandler := apirest.NewHandler(sessionManager, agentEngine, eventBus)
	sseHandler := sse.NewHandler(eventBus)

	server := gorest.MustNewServer(gorest.RestConf{
		Host: "0.0.0.0",
		Port: *port,
	})
	defer server.Stop()

	server.AddRoutes([]gorest.Route{
		{Method: "POST", Path: "/api/sessions", Handler: restHandler.CreateSession},
		{Method: "GET", Path: "/api/sessions", Handler: restHandler.ListSessions},
		{Method: "GET", Path: "/api/sessions/:id", Handler: restHandler.GetSession},
		{Method: "POST", Path: "/api/sessions/:id/input", Handler: restHandler.SubmitInput},
		{Method: "POST", Path: "/api/sessions/:id/approve", Handler: restHandler.ApprovePlan},
		{Method: "POST", Path: "/api/sessions/:id/unblock", Handler: restHandler.UnblockSession},
	})

	server.AddRoutes([]gorest.Route{
		{Method: "GET", Path: "/api/sessions/:id/events", Handler: sseHandler.StreamSessionEvents},
	}, gorest.WithSSE())

	log.Printf("REST+SSE server starting on port %d", *port)
	log.Printf("API endpoints:")
	log.Printf("  POST /api/sessions          - Create session")
	log.Printf("  GET  /api/sessions          - List sessions")
	log.Printf("  GET  /api/sessions/:id      - Get session")
	log.Printf("  POST /api/sessions/:id/input    - Submit input")
	log.Printf("  POST /api/sessions/:id/approve  - Approve plan")
	log.Printf("  POST /api/sessions/:id/unblock  - Unblock session")
	log.Printf("  GET  /api/sessions/:id/events   - SSE stream")

	server.Start()
}
