package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-base/internal/config"
	"agent-base/internal/engine"
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

	"github.com/sashabaranov/go-openai"
)

func main() {
	ctx := context.Background()

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

	// 6. 初始化子系统
	memoryMgr := memory.NewMemoryManager(filepath.Join(cfg.WorkDir, ".memory"))
	memoryMgr.LoadAll()

	tasksDir := filepath.Join(cfg.WorkDir, ".tasks")
	taskMgr := tasks.NewTaskManager(tasksDir)
	backgroundMgr := tasks.NewBackgroundManager(filepath.Join(cfg.WorkDir, ".runtime-tasks"))
	cronScheduler := tasks.NewCronScheduler()

	// 注册任务相关工具
	registry.Register(tasks.NewTaskCreateTool(taskMgr))
	registry.Register(tasks.NewTaskUpdateTool(taskMgr))
	registry.Register(tasks.NewTaskListTool(taskMgr))
	registry.Register(tasks.NewTaskGetTool(taskMgr))
	registry.Register(tasks.NewBackgroundRunTool(backgroundMgr))
	registry.Register(tasks.NewCheckBackgroundTool(backgroundMgr))
	registry.Register(tasks.NewCronCreateTool(cronScheduler))
	registry.Register(tasks.NewCronDeleteTool(cronScheduler))
	registry.Register(tasks.NewCronListTool(cronScheduler))

	// 注册 Memory 工具
	registry.Register(memory.NewSaveMemoryTool(memoryMgr))

	// 注册 Skills 工具
	skillLoader := skills.NewSkillLoader(filepath.Join(cfg.ProjectRoot, "skills"))
	skillLoader.LoadAll()
	registry.Register(skills.NewLoadSkillTool(skillLoader))

	// 注册 Subagent 工具
	subagentRunner := subagent.NewSubagentRunner(llmClient, registry, cfg.WorkDir, cfg.Model)
	registry.Register(subagent.NewTaskTool(subagentRunner))

	// 注册 Compact 工具
	registry.Register(engine.NewCompactTool())

	// 7. 初始化权限和 Hook 管理
	permissionMgr := security.NewPermissionManager("plan", cfg.WorkDir)
	hookMgr := events.NewHookManager(cfg.WorkDir, false)

	// 8. 创建引擎组件
	promptBuilder := engine.NewSystemPromptBuilder(cfg.WorkDir, cfg.ProjectRoot, cfg.Model)
	contextMgr := engine.NewContextManager(llmClient, cfg.Model, cfg.WorkDir, cfg.ContextThreshold)
	recoveryMgr := engine.NewRecoveryManager(llmClient, cfg.Model, contextMgr, promptBuilder)

	// 9. 组装 Agent 引擎
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

	// 10. 启动 Cron 调度器
	cronScheduler.Start()
	defer cronScheduler.Stop()

	// 11. 触发 SessionStart Hook
	hookMgr.RunHooks("SessionStart", map[string]interface{}{
		"tool_name":  "",
		"tool_input": map[string]interface{}{},
	})

	cmdProcessor := &CommandProcessor{
		permissionMgr: permissionMgr,
		taskMgr:       taskMgr,
		cronScheduler: cronScheduler,
		memoryMgr:     memoryMgr,
		promptBuilder: promptBuilder,
		contextMgr:    contextMgr,
	}

	// 12. 启动 REPL
	var history []openai.ChatCompletionMessage
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\033[32m[Agent MVP Ready - Refactored Architecture]\033[0m")
	fmt.Println("Commands: /mode <plan|build>, /tasks, /cron, /memories, /prompt, /compact")

	for {
		fmt.Print("\033[36magent >> \033[0m")
		query, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		query = strings.TrimSpace(query)
		if query == "" || strings.ToLower(query) == "q" || strings.ToLower(query) == "exit" {
			break
		}

		// 处理特殊命令
		isCmd, newHistory := cmdProcessor.Handle(query, history)
		history = newHistory
		if isCmd {
			continue
		}

		// 执行 Agent 循环
		// Drain background notifications
		bgNotifs := backgroundMgr.DrainNotifications()
		for _, notif := range bgNotifs {
			fmt.Printf("[bg:%s] %s: %s\n", notif.TaskID, notif.Status, notif.Preview)
			history = append(history, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("<background-results>\n[bg:%s] %s: %s (output_file=%s)\n</background-results>", notif.TaskID, notif.Status, notif.Preview, notif.OutputFile),
			})
		}

		// Drain cron notifications
		cronNotifs := cronScheduler.DrainNotifications()
		for _, notif := range cronNotifs {
			preview := notif
			if len(preview) > 100 {
				preview = preview[:100]
			}
			fmt.Printf("[Cron notification] %s\n", preview)
			history = append(history, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: notif,
			})
		}

		history = append(history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: query,
		})

		history, err = agentEngine.Run(ctx, history)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if len(history) > 0 {
			lastMsg := history[len(history)-1]
			if lastMsg.Role == openai.ChatMessageRoleAssistant && lastMsg.Content != "" {
				fmt.Println(lastMsg.Content)
			}
		}
		fmt.Println()
	}
}
