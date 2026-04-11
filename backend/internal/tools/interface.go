package tools

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) string
	Schema() openai.Tool
}

type ToolRegistry interface {
	Register(tool Tool)
	Get(name string) (Tool, bool)
	List() []Tool
	GetSchemas() []openai.Tool
	Execute(ctx context.Context, name string, args map[string]interface{}) (string, error)
}
