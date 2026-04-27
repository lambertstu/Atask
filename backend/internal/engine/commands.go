package engine

import (
	"fmt"
	"strings"

	"agent-base/internal/systems/memory"
	"agent-base/internal/systems/tasks"
	"agent-base/pkg/security"

	"github.com/sashabaranov/go-openai"
)

type CommandProcessor struct {
	PermissionMgr *security.PermissionManager
	TaskMgr       *tasks.TaskManager
	CronScheduler *tasks.CronScheduler
	MemoryMgr     *memory.MemoryManager
	PromptBuilder PromptBuilder
	ContextMgr    ContextManager
	Model         string
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
			cp.PermissionMgr.SetMode(mode)
			fmt.Printf("[Switched to %s mode]\n", mode)
		} else {
			fmt.Printf("Usage: /mode <%s>\n", strings.Join(modes, "|"))
		}
		return true, history
	}

	if query == "/tasks" {
		fmt.Println(cp.TaskMgr.ListAll())
		return true, history
	}

	if query == "/cron" {
		fmt.Println(cp.CronScheduler.ListTasks())
		return true, history
	}

	if query == "/memories" {
		memories := cp.MemoryMgr.ListMemories()
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
		fmt.Println(cp.PromptBuilder.Build())
		fmt.Println("--- End ---")
		return true, history
	}

	if query == "/compact" {
		fmt.Println("[Manual compact requested by user...]")
		history = cp.ContextMgr.AutoCompact(history, cp.Model)
		return true, history
	}

	fmt.Printf("Unknown command: %s. Type 'help' or check available commands.\n", query)
	return true, history
}
