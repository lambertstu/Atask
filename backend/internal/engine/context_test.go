package engine

import (
	"testing"

	"agent-base/testutil"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestNewContextManager(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	cm := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)

	assert.NotNil(t, cm)
}

func TestContextManager_EstimateTokens(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	cm := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "Hello"},
	}

	tokens := cm.EstimateTokens(messages)
	assert.Greater(t, tokens, 0)
}

func TestContextManager_MicroCompact(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	cm := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)

	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "Test"},
		{Role: openai.ChatMessageRoleTool, Content: "Long tool output that should be compacted"},
	}

	compactMsgs := cm.MicroCompact(messages)
	assert.LessOrEqual(t, len(compactMsgs), len(messages))
}
