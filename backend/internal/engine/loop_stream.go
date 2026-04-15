package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"agent-base/pkg/events"
	"agent-base/pkg/security"

	"github.com/sashabaranov/go-openai"
)

const UPDATE_STREAM = "<reminder>Update your todos.</reminder>"

func (e *AgentEngine) RunStream(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
	emitter EventEmitter,
	sessionID string,
) ([]openai.ChatCompletionMessage, error) {
	roundsSinceTodo := 0

	for {
		select {
		case <-ctx.Done():
			return messages, ctx.Err()
		default:
		}

		messages = e.contextMgr.MicroCompact(messages)

		if e.contextMgr.EstimateTokens(messages) > e.contextThreshold {
			emitter.Emit(events.EventThinking, map[string]interface{}{
				"session_id": sessionID,
				"action":     "auto_compact",
			})
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

		emitter.Emit(events.EventThinking, map[string]interface{}{
			"session_id": sessionID,
			"action":     "llm_call",
		})

		resp, err := e.recoveryMgr.CreateWithRecovery(ctx, req, &messages)
		if err != nil {
			emitter.Emit(events.EventError, map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return messages, err
		}

		assistantMessage := resp.Choices[0].Message
		messages = append(messages, assistantMessage)

		if len(assistantMessage.ToolCalls) == 0 {
			if assistantMessage.Content != "" {
				emitter.Emit(events.EventAssistantMessage, map[string]interface{}{
					"session_id": sessionID,
					"content":    assistantMessage.Content,
				})
			}
			return messages, nil
		}

		usedTodoThisRound := false
		manualCompactRequested := false

		var wg sync.WaitGroup
		var printMu sync.Mutex
		results := make([]string, len(assistantMessage.ToolCalls))
		errResults := make([]error, len(assistantMessage.ToolCalls))

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
					emitter.Emit(events.EventToolStart, map[string]interface{}{
						"session_id": sessionID,
						"tool_name":  toolCall.Function.Name,
						"args":       args,
						"status":     "denied",
					})
					emitter.Emit(events.EventToolEnd, map[string]interface{}{
						"session_id": sessionID,
						"tool_name":  toolCall.Function.Name,
						"output":     fmt.Sprintf("Permission denied: %s", reason),
					})
					return fmt.Sprintf("Permission denied: %s", reason)
				}
			} else if decision["behavior"] == "ask" {
				if e.handleAskStream(ctx, toolCall.Function.Name, args, decision, emitter, sessionID) {
					execFunc = func() string {
						return e.executeToolStream(toolCall, args, emitter, sessionID)
					}
				} else {
					execFunc = func() string {
						emitter.Emit(events.EventToolEnd, map[string]interface{}{
							"session_id": sessionID,
							"tool_name":  toolCall.Function.Name,
							"output":     fmt.Sprintf("Permission denied by user for %s", toolCall.Function.Name),
						})
						return fmt.Sprintf("Permission denied by user for %s", toolCall.Function.Name)
					}
				}
			} else {
				execFunc = func() string {
					return e.executeToolStream(toolCall, args, emitter, sessionID)
				}
			}

			preparedTools[index] = preparedTool{execFunc: execFunc}
		}

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

			if toolCall.Function.Name == "delegate_subagent" {
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
			emitter.Emit(events.EventThinking, map[string]interface{}{
				"session_id": sessionID,
				"action":     "manual_compact",
			})
			messages = e.contextMgr.AutoCompact(messages)
		}

		roundsSinceTodo++
		if usedTodoThisRound {
			roundsSinceTodo = 0
		}

		if roundsSinceTodo > 3 {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: UPDATE_STREAM,
			})
		}
	}
}

func (e *AgentEngine) handleAskStream(
	ctx context.Context,
	toolName string,
	toolInput map[string]interface{},
	decision map[string]interface{},
	emitter EventEmitter,
	sessionID string,
) bool {
	if e.permissionMgr.IsBlockingMode() {
		emitter.Emit(events.EventBlocked, map[string]interface{}{
			"session_id": sessionID,
			"tool_name":  toolName,
			"tool_input": toolInput,
			"decision":   decision,
			"blocked_on": "permission",
		})

		responseCh := make(chan security.BlockingResponse, 1)
		blockingReq := security.BlockingRequest{
			ToolName:   toolName,
			ToolInput:  toolInput,
			Decision:   decision,
			ResponseCh: responseCh,
		}

		if blockingChan := e.permissionMgr.GetBlockingChannel(); blockingChan != nil {
			select {
			case blockingChan <- blockingReq:
			case <-ctx.Done():
				return false
			}

			select {
			case resp := <-responseCh:
				if resp.Approved {
					if resp.AddAllowed != "" {
						e.permissionMgr.AddAllowedDir(resp.AddAllowed)
					}
					return true
				}
				return false
			case <-ctx.Done():
				return false
			}
		}
		return false
	}

	return e.permissionMgr.HandleAsk(toolName, toolInput, decision)
}

func (e *AgentEngine) executeToolStream(
	toolCall openai.ToolCall,
	args map[string]interface{},
	emitter EventEmitter,
	sessionID string,
) string {
	ctx := context.Background()

	emitter.Emit(events.EventToolStart, map[string]interface{}{
		"session_id": sessionID,
		"tool_name":  toolCall.Function.Name,
		"args":       args,
	})

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

	finalOutput := e.contextMgr.SaveLargeOutput(toolCall.Function.Name, output)

	emitter.Emit(events.EventToolEnd, map[string]interface{}{
		"session_id": sessionID,
		"tool_name":  toolCall.Function.Name,
		"output":     finalOutput,
	})

	return finalOutput
}
