package subagent

import (
	"context"
	"encoding/json"
	"fmt"

	"agent-base/internal/llm"
	"agent-base/internal/tools"

	"github.com/sashabaranov/go-openai"
)

const SubagentSystem = "You are a coding subagent. Complete the given task, then summarize your findings."

var ExcludedChildTools = map[string]bool{
	"task":           true, // 防止递归创建子代理
	"todo":           true, // 任务规划由主代理管理
	"task_create":    true, // 任务创建由主代理管理
	"task_update":    true, // 任务更新由主代理管理
	"background_run": true, // 后台任务由主代理控制
	"cron_create":    true, // 定时任务创建由主代理管理
	"cron_delete":    true, // 定时任务删除由主代理管理
}

type SubagentRunner struct {
	client   llm.LLMClient
	registry tools.ToolRegistry
	workDir  string
	model    string
	maxTurns int
}

func NewSubagentRunner(client llm.LLMClient, registry tools.ToolRegistry, workDir, model string) *SubagentRunner {
	return &SubagentRunner{
		client:   client,
		registry: registry,
		workDir:  workDir,
		model:    model,
		maxTurns: 30,
	}
}

func (sr *SubagentRunner) getChildTools() []openai.Tool {
	var childTools []openai.Tool
	allTools := sr.registry.GetSchemas()

	for _, tool := range allTools {
		if !ExcludedChildTools[tool.Function.Name] {
			childTools = append(childTools, tool)
		}
	}
	return childTools
}

func (sr *SubagentRunner) Run(ctx context.Context, prompt, description string) string {
	if description == "" {
		description = "subtask"
	}

	subMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	for i := 0; i < sr.maxTurns; i++ {
		req := openai.ChatCompletionRequest{
			Model: sr.model,
			Messages: append([]openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf("%s. You are at %s.", SubagentSystem, sr.workDir),
				},
			}, subMessages...),
			Tools:      sr.getChildTools(),
			ToolChoice: "auto",
		}

		resp, err := sr.client.CreateCompletion(ctx, req)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}

		assistantMessage := resp.Choices[0].Message
		subMessages = append(subMessages, assistantMessage)

		if len(assistantMessage.ToolCalls) == 0 {
			if assistantMessage.Content != "" {
				return assistantMessage.Content
			}
			return "(no summary)"
		}

		for _, toolCall := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				subMessages = append(subMessages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: toolCall.ID,
					Content:    fmt.Sprintf("Error: %v", err),
				})
				continue
			}

			output, err := sr.registry.Execute(ctx, toolCall.Function.Name, args)
			if err != nil {
				subMessages = append(subMessages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: toolCall.ID,
					Content:    err.Error(),
				})
				continue
			}

			if len(output) > 50000 {
				output = output[:50000]
			}

			subMessages = append(subMessages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: toolCall.ID,
				Content:    output,
			})
		}
	}

	return "(subagent exceeded max turns)"
}

type TaskTool struct {
	runner *SubagentRunner
}

func NewTaskTool(runner *SubagentRunner) tools.Tool {
	return &TaskTool{runner: runner}
}

func (t *TaskTool) Name() string {
	return "task"
}

func (t *TaskTool) Description() string {
	return "Delegate a subtask to a subagent with clean context."
}

func (t *TaskTool) Execute(ctx context.Context, args map[string]interface{}) string {
	prompt := ""
	description := ""

	if p, ok := args["prompt"].(string); ok {
		prompt = p
	}
	if d, ok := args["description"].(string); ok {
		description = d
	}

	if prompt == "" {
		return "Error: prompt is required"
	}

	return t.runner.Run(ctx, prompt, description)
}

func (t *TaskTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Task prompt for the subagent",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Brief description of the subtask",
					},
				},
				"required": []string{"prompt"},
			},
		},
	}
}
