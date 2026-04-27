package engine

import (
	"context"

	"agent-base/internal/llm"
	"agent-base/internal/tools"
	"agent-base/pkg/events"
	"agent-base/pkg/security"

	"github.com/sashabaranov/go-openai"
)

type EventEmitter interface {
	Emit(eventType events.EventType, data map[string]interface{})
}

type PromptBuilder interface {
	Build() string
}

type ContextManager interface {
	MicroCompact(messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage
	AutoCompact(messages []openai.ChatCompletionMessage, model string) []openai.ChatCompletionMessage
	EstimateTokens(messages []openai.ChatCompletionMessage) int
	SaveLargeOutput(toolName, output string) string
}

type RecoveryManager interface {
	CreateWithRecovery(ctx context.Context, req openai.ChatCompletionRequest, messages *[]openai.ChatCompletionMessage, emitter EventEmitter, sessionID string) (*openai.ChatCompletionResponse, error)
}

type AgentEngine struct {
	client           llm.LLMClient
	registry         tools.ToolRegistry
	permissionMgr    *security.PermissionManager
	hookMgr          *events.HookManager
	promptBuilder    PromptBuilder
	contextMgr       ContextManager
	recoveryMgr      RecoveryManager
	model            string
	contextThreshold int
}

func NewAgentEngine(
	client llm.LLMClient,
	registry tools.ToolRegistry,
	permissionMgr *security.PermissionManager,
	hookMgr *events.HookManager,
	promptBuilder PromptBuilder,
	contextMgr ContextManager,
	recoveryMgr RecoveryManager,
	model string,
	contextThreshold int,
) *AgentEngine {
	return &AgentEngine{
		client:           client,
		registry:         registry,
		permissionMgr:    permissionMgr,
		hookMgr:          hookMgr,
		promptBuilder:    promptBuilder,
		contextMgr:       contextMgr,
		recoveryMgr:      recoveryMgr,
		model:            model,
		contextThreshold: contextThreshold,
	}
}

func (e *AgentEngine) SetPermissionManager(pm *security.PermissionManager) {
	e.permissionMgr = pm
}

type CompactTool struct{}

func NewCompactTool() tools.Tool {
	return &CompactTool{}
}

func (t *CompactTool) Name() string {
	return "compact"
}

func (t *CompactTool) Description() string {
	return "Compact conversation history to free up context space."
}

func (t *CompactTool) Execute(ctx context.Context, args map[string]interface{}) string {
	return "Compact tool ready. Use engine to trigger compact."
}

func (t *CompactTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}
