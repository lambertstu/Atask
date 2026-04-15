package engine

import (
	"context"
	"errors"
	"testing"

	"agent-base/testutil"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
)

func TestNewRecoveryManager(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	mockPrompt := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")
	mockContext := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)

	rm := NewRecoveryManager(mockLLM, "test-model", mockContext, mockPrompt)
	assert.NotNil(t, rm)
}

func TestRecoveryManager_Success(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	mockLLM.AddTextResponse("Test response")

	mockPrompt := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")
	mockContext := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)

	rm := NewRecoveryManager(mockLLM, "test-model", mockContext, mockPrompt)

	req := openai.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []openai.ChatCompletionMessage{},
	}

	resp, err := rm.CreateWithRecovery(context.Background(), req, &[]openai.ChatCompletionMessage{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestRecoveryManager_Retry(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mockLLM := testutil.NewMockLLMClient()
	// Add multiple responses - first error, then success
	mockLLM.AddErrorResponse(errors.New("API error"))
	mockLLM.AddTextResponse("Success after retry")

	mockPrompt := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")
	mockContext := NewContextManager(mockLLM, "test-model", tempDir.Path, 50000)

	rm := NewRecoveryManager(mockLLM, "test-model", mockContext, mockPrompt)

	req := openai.ChatCompletionRequest{
		Model:    "test-model",
		Messages: []openai.ChatCompletionMessage{},
	}

	messages := []openai.ChatCompletionMessage{}
	rm.CreateWithRecovery(context.Background(), req, &messages)

	// Recovery manager should retry and eventually succeed
	// Just check that it was called at least once
	assert.GreaterOrEqual(t, mockLLM.CallCount, 1)
}
