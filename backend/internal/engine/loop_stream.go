package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"agent-base/internal/systems/session"
	"agent-base/pkg/events"

	"github.com/sashabaranov/go-openai"
)

const UPDATE_STREAM = "<reminder>Update your todos.</reminder>"

func (e *AgentEngine) RunStream(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
	emitter EventEmitter,
	sessionID string,
	sessionMgr *session.SessionManager,
) ([]openai.ChatCompletionMessage, error) {
	roundsSinceTodo := 0

	currentModel := e.model
	if sessionMgr != nil {
		if sess := sessionMgr.GetSession(sessionID); sess != nil && sess.Model != "" {
			currentModel = sess.Model
		}
	}

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
			messages = e.contextMgr.AutoCompact(messages, currentModel)
		}

		system := e.promptBuilder.Build()

		req := openai.ChatCompletionRequest{
			Model: currentModel,
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

		resp, err := e.recoveryMgr.CreateWithRecovery(ctx, req, &messages, emitter, sessionID)
		if err != nil {
			emitter.Emit(events.EventError, map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return messages, err
		}

		assistantMessage := resp.Choices[0].Message
		messages = append(messages, assistantMessage)
		if sessionMgr != nil {
			sessionMgr.UpdateMessages(sessionID, messages)
		}

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

		preparedExecFuncs := make([]func() string, len(assistantMessage.ToolCalls))

		// 执行llm 返回的工具
		for i, tc := range assistantMessage.ToolCalls {
			index := i
			toolCall := tc

			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				errResults[index] = err
				continue
			}

			// 权限校验
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
				if e.handleAskStream(toolCall.Function.Name, args, decision, emitter, sessionID, sessionMgr) {
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

			preparedExecFuncs[index] = execFunc
		}

		for i, tc := range assistantMessage.ToolCalls {
			index := i
			toolCall := tc
			execFunc := preparedExecFuncs[index]

			if errResults[index] != nil {
				continue
			}

			runTool := func(idx int, call openai.ToolCall, fn func() string) {
				output := fn()

				printMu.Lock()
				//fmt.Printf("\033[33m> %s %s:\033[0m\n", call.Function.Name, call.Function.Arguments)
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
				}(index, toolCall, execFunc)
			} else {
				runTool(index, toolCall, execFunc)
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
			if sessionMgr != nil {
				sessionMgr.UpdateMessages(sessionID, messages)
			}

			if toolCall.Function.Name == "todo" {
				usedTodoThisRound = true
			}

			if toolCall.Function.Name == "compact" {
				manualCompactRequested = true
			}
		}

		if manualCompactRequested {
			messages = e.contextMgr.AutoCompact(messages, currentModel)
			if sessionMgr != nil {
				sessionMgr.UpdateMessages(sessionID, messages)
			}
			emitter.Emit(events.EventThinking, map[string]interface{}{
				"session_id": sessionID,
				"action":     "manual_compact",
			})
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
	toolName string,
	toolInput map[string]interface{},
	decision map[string]interface{},
	emitter EventEmitter,
	sessionID string,
	sessionMgr *session.SessionManager,
) bool {
	if e.permissionMgr.IsBlockingMode() {
		emitter.Emit(events.EventBlocked, map[string]interface{}{
			"session_id": sessionID,
			"tool_name":  toolName,
			"tool_input": toolInput,
			"decision":   decision,
			"blocked_on": "permission",
		})

		sessionMgr.SetBlocked(sessionID, "permission", toolName, toolInput)

		sess := sessionMgr.GetSession(sessionID)
		if sess == nil {
			return false
		}

		select {
		case <-sess.Ctx.Done():
			return false
		case res := <-sess.BlockedResponse:
			if res.Approved {
				if res.AddAllowed != "" {
					e.permissionMgr.AddAllowedDir(res.AddAllowed)
				}
				return true
			}
			return false
		}
	}

	return e.permissionMgr.AskUserREPL(toolName, toolInput, decision)
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
