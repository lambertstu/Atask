package engine

import (
	"context"
	"testing"

	"agent-base/internal/tools"
	"agent-base/internal/tools/builtin"
	"agent-base/pkg/events"
	"agent-base/pkg/security"
	"agent-base/testutil"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentEngine(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	registry := tools.NewDefaultRegistry()
	pm := security.NewPermissionManager("default", tempDir.Path)
	hm := events.NewHookManager(tempDir.Path)
	builder := NewSystemPromptBuilder(tempDir.Path, "test-model")
	cm := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)
	rm := NewRecoveryManager(mockLLM, "test-model", cm, builder)

	engine := NewAgentEngine(mockLLM, registry, pm, hm, builder, cm, rm, "test-model", 50000)
	assert.NotNil(t, engine)
}

func TestAgentEngine_Run_NoToolCalls(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	mockLLM.AddTextResponse("Hello! I'm ready to help.")

	registry := tools.NewDefaultRegistry()
	pm := security.NewPermissionManager("auto", tempDir.Path)
	hm := events.NewHookManager(tempDir.Path)
	builder := NewSystemPromptBuilder(tempDir.Path, "test-model")
	cm := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)
	rm := NewRecoveryManager(mockLLM, "test-model", cm, builder)

	engine := NewAgentEngine(mockLLM, registry, pm, hm, builder, cm, rm, "test-model", 50000)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "Hello"},
	}

	result, err := engine.Run(context.Background(), messages)
	assert.NoError(t, err)
	assert.Greater(t, len(result), 1)
}

func TestAgentEngine_ExecuteTool(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	registry := tools.NewDefaultRegistry()
	registry.Register(builtin.NewReadTool(tempDir.Path))

	pm := security.NewPermissionManager("auto", tempDir.Path)
	hm := events.NewHookManager(tempDir.Path)
	builder := NewSystemPromptBuilder(tempDir.Path, "test-model")
	cm := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)
	rm := NewRecoveryManager(mockLLM, "test-model", cm, builder)

	engine := NewAgentEngine(mockLLM, registry, pm, hm, builder, cm, rm, "test-model", 50000)

	// Create test file
	testFile := tempDir.CreateFile("test.txt", "Hello World")

	// Execute tool via registry
	output := engine.executeTool(openai.ToolCall{
		ID: "test-id",
		Function: openai.FunctionCall{
			Name:      "read_file",
			Arguments: `{"path": "` + testFile + `"}`,
		},
	}, map[string]interface{}{
		"path":         testFile,
		"allowed_dirs": []string{},
	})

	assert.Contains(t, output, "Hello World")
}

func TestCompactTool(t *testing.T) {
	tool := NewCompactTool()
	assert.Equal(t, "compact", tool.Name())

	result := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Contains(t, result, "Compact")
}
