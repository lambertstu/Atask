package engine

import (
	"testing"

	"agent-base/internal/config"
	"agent-base/internal/llm"
	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestEngineManager_GetOrCreate(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	cfg := config.Config{
		ProjectRoot:      tempDir.Path,
		Model:            "test-model",
		ContextThreshold: 50000,
		BashTimeout:      30,
		APIKey:           "test-api-key",
		BaseURL:          "https://test.example.com/v1",
	}

	llmClient := llm.NewClient(&cfg)
	em := NewEngineManager(cfg, llmClient)

	engCtx := em.GetOrCreate(tempDir.Path)
	assert.NotNil(t, engCtx)
	assert.NotNil(t, engCtx.Engine)
	assert.NotNil(t, engCtx.Registry)
	assert.NotNil(t, engCtx.HookMgr)
	assert.NotNil(t, engCtx.PromptBuilder)
	assert.NotNil(t, engCtx.ContextMgr)
	assert.NotNil(t, engCtx.RecoveryMgr)
}

func TestEngineManager_GetOrCreate_Cached(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	cfg := config.Config{
		ProjectRoot:      tempDir.Path,
		Model:            "test-model",
		ContextThreshold: 50000,
		BashTimeout:      30,
		APIKey:           "test-api-key",
		BaseURL:          "https://test.example.com/v1",
	}

	llmClient := llm.NewClient(&cfg)
	em := NewEngineManager(cfg, llmClient)

	engCtx1 := em.GetOrCreate(tempDir.Path)
	engCtx2 := em.GetOrCreate(tempDir.Path)

	assert.Equal(t, engCtx1, engCtx2)
}

func TestEngineManager_Get(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	cfg := config.Config{
		ProjectRoot:      tempDir.Path,
		Model:            "test-model",
		ContextThreshold: 50000,
		BashTimeout:      30,
		APIKey:           "test-api-key",
		BaseURL:          "https://test.example.com/v1",
	}

	llmClient := llm.NewClient(&cfg)
	em := NewEngineManager(cfg, llmClient)

	engCtx, exists := em.Get(tempDir.Path)
	assert.False(t, exists)
	assert.Nil(t, engCtx)

	em.GetOrCreate(tempDir.Path)
	engCtx, exists = em.Get(tempDir.Path)
	assert.True(t, exists)
	assert.NotNil(t, engCtx)
}
