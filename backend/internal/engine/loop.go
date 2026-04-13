package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
		manualCompactRequested := false

		var wg sync.WaitGroup
		var printMu sync.Mutex
		results := make([]string, len(assistantMessage.ToolCalls))
		errResults := make([]error, len(assistantMessage.ToolCalls))

		// 所有工具都在主线程进行权限检查和交互询问（避免并发访问 permissionMgr 和终端错乱）
		type preparedTool struct {
			execFunc func() string
		}
		preparedTools := make([]preparedTool, len(assistantMessage.ToolCalls))

		for i, tc := range assistantMessage.ToolCalls {
			index := i
			toolCall := tc

			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				errResults[index] = err
				continue
			}

			decision := e.permissionMgr.Check(toolCall.Function.Name, args)

			var execFunc func() string
			if decision["behavior"] == "deny" {
				reason := decision["reason"]
				execFunc = func() string {
					printMu.Lock()
					defer printMu.Unlock()
					fmt.Printf("\033[31m[DENIED] %s: %s\033[0m\n", toolCall.Function.Name, reason)
					return fmt.Sprintf("Permission denied: %s", reason)
				}
			} else if decision["behavior"] == "ask" {
				if e.permissionMgr.HandleAsk(toolCall.Function.Name, args, decision) {
					execFunc = func() string {
						return e.executeTool(toolCall, args)
					}
				} else {
					execFunc = func() string {
						printMu.Lock()
						defer printMu.Unlock()
						fmt.Printf("\033[31m[USER DENIED] %s\033[0m\n", toolCall.Function.Name)
						return fmt.Sprintf("Permission denied by user for %s", toolCall.Function.Name)
					}
				}
			} else {
				execFunc = func() string {
					return e.executeTool(toolCall, args)
				}
			}

			preparedTools[index] = preparedTool{execFunc: execFunc}
		}

		// 只有 task 工具放入协程异步执行，其它工具仍在主线程同步执行
		for i, tc := range assistantMessage.ToolCalls {
			index := i
			toolCall := tc
			prep := preparedTools[index]

			if errResults[index] != nil {
				continue
			}

			runTool := func(idx int, call openai.ToolCall, fn func() string) {
				output := fn()

				printMu.Lock()
				fmt.Printf("\033[33m> %s %s:\033[0m\n", call.Function.Name, call.Function.Arguments)
				if len(output) > 200 {
					fmt.Println(output[:200])
				} else {
					fmt.Println(output)
				}
				printMu.Unlock()

				results[idx] = output
			}

			if toolCall.Function.Name == "task" {
				wg.Add(1)
				go func(idx int, call openai.ToolCall, fn func() string) {
					defer wg.Done()
					runTool(idx, call, fn)
				}(index, toolCall, prep.execFunc)
			} else {
				runTool(index, toolCall, prep.execFunc)
			}
		}

		wg.Wait()

		for i, toolCall := range assistantMessage.ToolCalls {
			var finalOutput string
			if errResults[i] != nil {
				finalOutput = fmt.Sprintf("Error: %v", errResults[i])
			} else {
				finalOutput = results[i]
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: toolCall.ID,
				Content:    finalOutput,
			})

			if toolCall.Function.Name == "todo" {
				usedTodoThisRound = true
			}

			if toolCall.Function.Name == "compact" {
				manualCompactRequested = true
			}
		}

		if manualCompactRequested {
			fmt.Println("[manual compact triggered]")
			messages = e.contextMgr.AutoCompact(messages)
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

	return e.contextMgr.SaveLargeOutput(toolCall.Function.Name, output)
}
