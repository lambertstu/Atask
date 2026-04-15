package engine

import (
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewSystemPromptBuilder(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	builder := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")
	assert.NotNil(t, builder)
}

func TestPromptBuilder_Build(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	builder := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")

	prompt := builder.Build()
	assert.Contains(t, prompt, "agent")
	assert.Contains(t, prompt, "test-model")
}

func TestPromptBuilder_BuildCore(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	builder := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")

	core := builder.buildCore()
	assert.Contains(t, core, "agent")
}

func TestPromptBuilder_BuildDynamicContext(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	builder := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")

	dynamic := builder.buildDynamicContext()
	assert.Contains(t, dynamic, "Working directory")
	assert.Contains(t, dynamic, "model")
}

func TestPromptBuilder_WithMemory(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	// Create memory directory
	memDir := tempDir.Subdir(".memory")
	memContent := `---
name: test
type: user
description: Test memory
---
Test memory content.`
	memDir.CreateFile("test.md", memContent)

	builder := NewSystemPromptBuilder(tempDir.Path, tempDir.Path, "test-model")

	prompt := builder.Build()
	assert.Contains(t, prompt, "agent")
}
