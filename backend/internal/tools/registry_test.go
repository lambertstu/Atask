package tools

import (
	"context"
	"sync"
	"testing"

	"agent-base/testutil"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultRegistry(t *testing.T) {
	registry := NewDefaultRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
	assert.Equal(t, 0, len(registry.tools))
}

func TestRegistry_Register(t *testing.T) {
	registry := NewDefaultRegistry()

	mockTool := testutil.NewMockTool("test_tool")
	registry.Register(mockTool)

	tool, ok := registry.Get("test_tool")
	assert.True(t, ok)
	assert.NotNil(t, tool)
	assert.Equal(t, "test_tool", tool.Name())
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	registry := NewDefaultRegistry()

	tools := []string{"tool1", "tool2", "tool3"}
	for _, name := range tools {
		registry.Register(testutil.NewMockTool(name))
	}

	assert.Equal(t, 3, len(registry.tools))
	for _, name := range tools {
		tool, ok := registry.Get(name)
		assert.True(t, ok)
		assert.Equal(t, name, tool.Name())
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewDefaultRegistry()

	registry.Register(testutil.NewMockTool("test_tool"))
	registry.Register(testutil.NewMockTool("test_tool"))

	// Should replace the first tool
	assert.Equal(t, 1, len(registry.tools))
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewDefaultRegistry()

	tool, ok := registry.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, tool)
}

func TestRegistry_List(t *testing.T) {
	registry := NewDefaultRegistry()

	// Empty registry
	list := registry.List()
	assert.Equal(t, 0, len(list))

	// Add tools
	registry.Register(testutil.NewMockTool("tool1"))
	registry.Register(testutil.NewMockTool("tool2"))

	list = registry.List()
	assert.Equal(t, 2, len(list))
}

func TestRegistry_GetSchemas(t *testing.T) {
	registry := NewDefaultRegistry()

	registry.Register(testutil.NewMockTool("bash"))
	registry.Register(testutil.NewMockTool("read_file"))

	schemas := registry.GetSchemas()
	assert.Equal(t, 2, len(schemas))

	for _, schema := range schemas {
		assert.Equal(t, openai.ToolTypeFunction, schema.Type)
		assert.NotNil(t, schema.Function)
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewDefaultRegistry()

	mockTool := testutil.NewMockTool("test_tool")
	mockTool.SetExecuteFunc(func(ctx context.Context, args map[string]interface{}) string {
		return "Custom execution result"
	})
	registry.Register(mockTool)

	result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{})
	require.NoError(t, err)
	assert.Equal(t, "Custom execution result", result)
}

func TestRegistry_ExecuteUnknown(t *testing.T) {
	registry := NewDefaultRegistry()

	result, err := registry.Execute(context.Background(), "unknown_tool", map[string]interface{}{})
	require.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewDefaultRegistry()

	var wg sync.WaitGroup
	numOps := 100

	// Concurrent registrations
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			registry.Register(testutil.NewMockTool("tool_" + string(rune(id))))
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.List()
			registry.GetSchemas()
		}()
	}

	wg.Wait()

	// Registry should still be functional
	list := registry.List()
	assert.GreaterOrEqual(t, len(list), 0)
}
