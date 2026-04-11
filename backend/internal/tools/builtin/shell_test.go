package builtin

import (
	"context"
	"testing"
	"time"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewBashTool(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)
	assert.NotNil(t, tool)
	assert.Equal(t, "bash", tool.Name())
	assert.Equal(t, "Run a shell command.", tool.Description())
	assert.Equal(t, 120*time.Second, tool.timeout)
	assert.Equal(t, 50000, tool.maxOutput)
}

func TestBashTool_ExecuteBasic(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	tests := []struct {
		command  string
		contains string
	}{
		{"echo hello", "hello"},
		{"ls", ""},
		{"pwd", tempDir.Path},
		{"echo 'test'", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]interface{}{"command": tt.command})
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
			} else {
				assert.NotContains(t, result, "Error:")
			}
		})
	}
}

func TestBashTool_ExecuteEmptyOutput(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	// Command with no output
	result := tool.Execute(context.Background(), map[string]interface{}{"command": "true"})
	assert.Equal(t, "(no output)", result)
}

func TestBashTool_ExecuteEmptyCommand(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	result := tool.Execute(context.Background(), map[string]interface{}{"command": ""})
	assert.Contains(t, result, "Error: command is required")
}

func TestBashTool_ExecuteDangerous(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	tests := []string{
		"rm -rf /",
		"sudo ls",
		"shutdown",
		"reboot",
		"echo test > /dev/null",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			result := tool.Execute(context.Background(), map[string]interface{}{"command": cmd})
			assert.Contains(t, result, "Error: Dangerous command blocked")
		})
	}
}

func TestBashTool_ExecuteLongOutput(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	// Generate long output
	result := tool.Execute(context.Background(), map[string]interface{}{
		"command": "for i in {1..10000}; do echo 'line'$i; done",
	})

	// Should be truncated to maxOutput
	assert.LessOrEqual(t, len(result), tool.maxOutput+100) // Allow some buffer
}

func TestBashTool_ExecuteWriteFile(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	// Write a file using bash
	_ = tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo 'test content' > test.txt",
	})

	// Verify file was created
	assert.True(t, tempDir.Exists("test.txt"))
	content := tempDir.ReadFile("test.txt")
	assert.Contains(t, content, "test content")
}

func TestBashTool_ExecuteWithExitCode(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	// Command that fails with exit code 1
	result := tool.Execute(context.Background(), map[string]interface{}{
		"command": "ls nonexistent",
	})

	assert.Contains(t, result, "No such file")
}

func TestBashTool_Schema(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	schema := tool.Schema()
	assert.Equal(t, "bash", schema.Function.Name)
	assert.NotNil(t, schema.Function.Parameters)
}

func TestBashTool_ExecuteConcurrent(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewBashTool(tempDir.Path, 120)

	done := make(chan bool, 5)

	// Run multiple commands concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			_ = tool.Execute(context.Background(), map[string]interface{}{
				"command": "echo concurrent_" + string(rune(id+'0')),
			})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
