package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/sashabaranov/go-openai"
)

type DefaultRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *DefaultRegistry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

func (r *DefaultRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *DefaultRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Tool
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

func (r *DefaultRegistry) GetSchemas() []openai.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var schemas []openai.Tool
	for _, tool := range r.tools {
		schemas = append(schemas, tool.Schema())
	}
	return schemas
}

func (r *DefaultRegistry) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	tool, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(ctx, args), nil
}
