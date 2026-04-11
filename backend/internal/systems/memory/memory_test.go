package memory

import (
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewMemoryManager(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mm := NewMemoryManager(tempDir.Path)
	assert.NotNil(t, mm)
}

func TestMemoryManager_SaveMemory(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mm := NewMemoryManager(tempDir.Path)

	result := mm.SaveMemory("test-memory", "Test memory description", "user", "Test content")
	assert.NotContains(t, result, "Error")
	assert.True(t, tempDir.Exists("test-memory.md"))
}

func TestMemoryManager_SaveMemoryInvalidType(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mm := NewMemoryManager(tempDir.Path)

	result := mm.SaveMemory("test", "Desc", "invalid_type", "Content")
	assert.Contains(t, result, "Error")
}

func TestMemoryManager_ListMemories(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	mm := NewMemoryManager(tempDir.Path)

	mm.SaveMemory("mem1", "Memory 1", "user", "Content 1")
	mm.LoadAll()

	memories := mm.ListMemories()
	assert.NotNil(t, memories)
}

func TestMemoryManager_LoadAll(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	// Pre-create a memory file
	content := `---
name: test
type: user
description: Test memory
---
Test memory content.`
	tempDir.CreateFile("test.md", content)

	mm := NewMemoryManager(tempDir.Path)
	mm.LoadAll()

	memories := mm.ListMemories()
	assert.NotNil(t, memories)
}
