package testutil

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type MockResponse struct {
	Content      string
	ToolCalls    []openai.ToolCall
	FinishReason string
	Error        error
}

type MockLLMClient struct {
	Responses   []MockResponse
	CallCount   int
	LastRequest openai.ChatCompletionRequest
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		Responses: []MockResponse{},
	}
}

func (m *MockLLMClient) CreateCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	m.CallCount++
	m.LastRequest = req

	if m.CallCount <= len(m.Responses) {
		resp := m.Responses[m.CallCount-1]
		if resp.Error != nil {
			return openai.ChatCompletionResponse{}, resp.Error
		}

		message := openai.ChatCompletionMessage{
			Role:      openai.ChatMessageRoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}

		finishReason := openai.FinishReasonStop
		if resp.FinishReason != "" {
			finishReason = openai.FinishReason(resp.FinishReason)
		}

		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message:      message,
					FinishReason: finishReason,
				},
			},
		}, nil
	}

	// Default response if no more predefined responses
	return openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "Default mock response",
				},
				FinishReason: openai.FinishReasonStop,
			},
		},
	}, nil
}

func (m *MockLLMClient) AddResponse(resp MockResponse) {
	m.Responses = append(m.Responses, resp)
}

func (m *MockLLMClient) AddTextResponse(content string) {
	m.AddResponse(MockResponse{Content: content})
}

func (m *MockLLMClient) AddToolCallResponse(toolCalls []openai.ToolCall) {
	m.AddResponse(MockResponse{ToolCalls: toolCalls})
}

func (m *MockLLMClient) AddErrorResponse(err error) {
	m.AddResponse(MockResponse{Error: err})
}

func (m *MockLLMClient) Reset() {
	m.CallCount = 0
	m.Responses = []MockResponse{}
	m.LastRequest = openai.ChatCompletionRequest{}
}

type MockTool struct {
	name        string
	description string
	schema      openai.Tool
	executeFunc func(ctx context.Context, args map[string]interface{}) string
}

func NewMockTool(name string) *MockTool {
	return &MockTool{
		name:        name,
		description: "Mock tool for testing",
		schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: "Mock tool for testing",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

func (t *MockTool) Name() string {
	return t.name
}

func (t *MockTool) Description() string {
	return t.description
}

func (t *MockTool) Execute(ctx context.Context, args map[string]interface{}) string {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, args)
	}
	return "Mock tool executed successfully"
}

func (t *MockTool) Schema() openai.Tool {
	return t.schema
}

func (t *MockTool) SetExecuteFunc(f func(ctx context.Context, args map[string]interface{}) string) {
	t.executeFunc = f
}
