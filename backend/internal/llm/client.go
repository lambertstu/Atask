package llm

import (
	"context"

	"agent-base/internal/config"
	"github.com/sashabaranov/go-openai"
)

type LLMClient interface {
	CreateCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

type DashScopeClient struct {
	client *openai.Client
	model  string
}

func NewClient(cfg *config.Config) *DashScopeClient {
	openaiConfig := openai.DefaultConfig(cfg.APIKey)
	openaiConfig.BaseURL = cfg.BaseURL

	return &DashScopeClient{
		client: openai.NewClientWithConfig(openaiConfig),
		model:  cfg.Model,
	}
}

func (c *DashScopeClient) CreateCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	return c.client.CreateChatCompletion(ctx, req)
}

func (c *DashScopeClient) GetModel() string {
	return c.model
}
