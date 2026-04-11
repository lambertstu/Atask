package events

import (
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewHookManager(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	hm := NewHookManager(tempDir.Path, false)
	assert.NotNil(t, hm)
}

func TestHookManager_RunHooks(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	hm := NewHookManager(tempDir.Path, false)

	result := hm.RunHooks("PreToolUse", map[string]interface{}{
		"tool_name":  "test",
		"tool_input": map[string]interface{}{},
	})

	assert.False(t, result.Blocked)
}

func TestHookManager_WithConfig(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	// Create hooks config
	tempDir.CreateFile(".hooks.json", `{
		"hooks": {
			"PreToolUse": []
		}
	}`)

	hm := NewHookManager(tempDir.Path, false)
	assert.NotNil(t, hm)
}

func TestHookResult(t *testing.T) {
	result := HookResult{
		Blocked:     false,
		BlockReason: "",
		Messages:    []string{},
	}

	assert.False(t, result.Blocked)
	assert.Empty(t, result.Messages)
}
