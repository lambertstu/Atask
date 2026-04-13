package main

import (
	"fmt"
	"strings"

	"agent-base/internal/engine"
	"agent-base/internal/systems/memory"
	"agent-base/internal/systems/tasks"
	"agent-base/pkg/security"

	"github.com/sashabaranov/go-openai"
)

type CommandProcessor struct {
	permissionMgr *security.PermissionManager
	taskMgr       *tasks.TaskManager
	cronScheduler *tasks.CronScheduler
	memoryMgr     *memory.MemoryManager
	promptBuilder engine.PromptBuilder
	contextMgr    engine.ContextManager
}

// Handle 处理输入的命令，返回 (是否是系统命令, 更新后的history)
func (cp *CommandProcessor) Handle(query string, history []openai.ChatCompletionMessage) (bool, []openai.ChatCompletionMessage) {
	if !strings.HasPrefix(query, "/") {
		return false, history
	}

	if strings.HasPrefix(query, "/mode ") {
		mode := strings.TrimSpace(strings.TrimPrefix(query, "/mode "))
		modes := []string{"plan", "build"}
		valid := false
		for _, m := range modes {
			if m == mode {
				valid = true
				break
			}
		}
		if valid {
			cp.permissionMgr.SetMode(mode)
			fmt.Printf("[Switched to %s mode]\n", mode)
		} else {
			fmt.Printf("Usage: /mode <%s>\n", strings.Join(modes, "|"))
		}
		return true, history
	}

	if query == "/tasks" {
		fmt.Println(cp.taskMgr.ListAll())
		return true, history
	}

	if query == "/cron" {
		fmt.Println(cp.cronScheduler.ListTasks())
		return true, history
	}

	if query == "/memories" {
		memories := cp.memoryMgr.ListMemories()
		if len(memories) > 0 {
			for name, mem := range memories {
				fmt.Printf("  [%s] %s: %s\n", mem.Type, name, mem.Description)
			}
		} else {
			fmt.Println("  (no memories)")
		}
		return true, history
	}

	if query == "/prompt" {
		fmt.Println("--- System Prompt ---")
		fmt.Println(cp.promptBuilder.Build())
		fmt.Println("--- End ---")
		return true, history
	}

	if query == "/compact" {
		fmt.Println("[Manual compact requested by user...]")
		history = cp.contextMgr.AutoCompact(history)
		return true, history
	}

	fmt.Printf("Unknown command: %s. Type 'help' or check available commands.\n", query)
	return true, history
}
