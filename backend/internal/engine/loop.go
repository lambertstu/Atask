package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

const UPDATE = "<reminder>Update your todos.</reminder>"

func (e *AgentEngine) Run(ctx context.Context, messages []openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, error) {
	roundsSinceTodo := 0

	for {
		// Micro compact
		messages = e.contextMgr.MicroCompact(messages)

		if e.contextMgr.EstimateTokens(messages) > e.contextThreshold {
			fmt.Println("[auto_compact triggered]")
			messages = e.contextMgr.AutoCompact(messages)
		}

		system := e.promptBuilder.Build()

		req := openai.ChatCompletionRequest{
			Model: e.model,
			Messages: append([]openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: system,
				},
			}, messages...),
			Tools:      e.registry.GetSchemas(),
			ToolChoice: "auto",
		}

		resp, err := e.recoveryMgr.CreateWithRecovery(ctx, req, &messages)
		if err != nil {
			return messages, err
		}

		assistantMessage := resp.Choices[0].Message
		messages = append(messages, assistantMessage)

		if len(assistantMessage.ToolCalls) == 0 {
			return messages, nil
		}

		usedTodoThisRound := false
		for _, toolCall := range assistantMessage.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: toolCall.ID,
					Content:    fmt.Sprintf("Error: %v", err),
				})
				continue
			}

			decision := e.permissionMgr.Check(toolCall.Function.Name, args)
			var output string

			if decision["behavior"] == "deny" {
				output = fmt.Sprintf("Permission denied: %s", decision["reason"])
				fmt.Printf("\033[31m[DENIED] %s: %s\033[0m\n", toolCall.Function.Name, decision["reason"])
			} else if decision["behavior"] == "ask" {
				if decision["needs_path_auth"] == true {
					requestedPath := decision["requested_path"].(string)
					requestedDir := filepath.Dir(requestedPath)

					fmt.Printf("\033[33m[PATH AUTH]\033[0m Grant access to directory: %s? (y/n): ", requestedDir)
					var response string
					fmt.Scanln(&response)

					if strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
						e.permissionMgr.AddAllowedDir(requestedDir)
						fmt.Printf("\033[32m[AUTHORIZED]\033[0m %s\n", requestedDir)
						output = e.executeTool(toolCall, args)
					} else {
						output = fmt.Sprintf("Path access denied: %s", requestedPath)
						fmt.Printf("\033[31m[PATH DENIED]\033[0m %s\n", requestedPath)
					}
				} else {
					if e.permissionMgr.AskUser(toolCall.Function.Name, args) {
						output = e.executeTool(toolCall, args)
					} else {
						output = fmt.Sprintf("Permission denied by user for %s", toolCall.Function.Name)
						fmt.Printf("\033[31m[USER DENIED] %s\033[0m\n", toolCall.Function.Name)
					}
				}
			} else {
				output = e.executeTool(toolCall, args)
			}

			fmt.Printf("\033[33m> %s %s:\033[0m\n", toolCall.Function.Name, toolCall.Function.Arguments)
			if len(output) > 200 {
				fmt.Println(output[:200])
			} else {
				fmt.Println(output)
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: toolCall.ID,
				Content:    output,
			})

			if toolCall.Function.Name == "todo" {
				usedTodoThisRound = true
			}

			if toolCall.Function.Name == "compact" {
				fmt.Println("[manual compact]")
				messages = e.contextMgr.AutoCompact(messages)
			}
		}

		roundsSinceTodo++
		if usedTodoThisRound {
			roundsSinceTodo = 0
		}

		if roundsSinceTodo > 3 {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: UPDATE,
			})
		}
	}
}

func (e *AgentEngine) executeTool(toolCall openai.ToolCall, args map[string]interface{}) string {
	ctx := context.Background()

	// 注入 allowedDirs 到 args（用于文件工具的路径检查）
	args["allowed_dirs"] = e.permissionMgr.GetAllowedDirs()

	hookCtx := map[string]interface{}{
		"tool_name":  toolCall.Function.Name,
		"tool_input": args,
	}
	preResult := e.hookMgr.RunHooks("PreToolUse", hookCtx)

	var output string
	if preResult.Blocked {
		output = fmt.Sprintf("Tool blocked by PreToolUse hook: %s", preResult.BlockReason)
	} else {
		tool, ok := e.registry.Get(toolCall.Function.Name)
		if !ok {
			output = fmt.Sprintf("Unknown tool: %s", toolCall.Function.Name)
		} else {
			output = tool.Execute(ctx, args)
		}

		hookCtx["tool_output"] = output
		postResult := e.hookMgr.RunHooks("PostToolUse", hookCtx)
		for _, msg := range postResult.Messages {
			output += fmt.Sprintf("\n[Hook note]: %s", msg)
		}
	}

	if len(output) > 50000 {
		output = utils.Truncate(output, 50000)
	}

	return output
}
