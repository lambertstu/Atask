package llm

import (
	"context"
	"sync"

	"github.com/sashabaranov/go-openai"
)

type LLMClient interface {
	CreateCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

type DashScopeClient struct {
	mu     sync.RWMutex
	client *openai.Client
	model  string
}

func NewClient(apiKey, baseURL, model string) *DashScopeClient {
	openaiConfig := openai.DefaultConfig(apiKey)
	openaiConfig.BaseURL = baseURL

	return &DashScopeClient{
		client: openai.NewClientWithConfig(openaiConfig),
		model:  model,
	}
}

func (c *DashScopeClient) CreateCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.client.CreateChatCompletion(ctx, req)
}

func (c *DashScopeClient) GetModel() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.model
}

func (c *DashScopeClient) UpdateConfig(apiKey, baseURL, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	openaiConfig := openai.DefaultConfig(apiKey)
	openaiConfig.BaseURL = baseURL
	c.client = openai.NewClientWithConfig(openaiConfig)
	c.model = model
}
