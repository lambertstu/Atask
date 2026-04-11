package builtin

import (
	"context"
	"fmt"
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewReadTool(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)
	assert.NotNil(t, tool)
	assert.Equal(t, "read_file", tool.Name())
	assert.Equal(t, 50000, tool.maxOutput)
}

func TestReadTool_ExecuteBasic(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	// Create test file
	testContent := "Hello, World!\nThis is a test file."
	testFile := tempDir.CreateFile("test.txt", testContent)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": testFile,
	})

	assert.Contains(t, result, "Hello, World!")
	assert.Contains(t, result, "test file")
}

func TestReadTool_ExecuteNotFound(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": "nonexistent.txt",
	})

	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "not found")
}

func TestReadTool_ExecuteEmptyPath(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": "",
	})

	assert.Contains(t, result, "Error: path is required")
}

func TestReadTool_ExecuteWithPathEscape(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": "/etc/passwd",
	})

	assert.Contains(t, result, "PATH_AUTH_REQUIRED")
}

func TestReadTool_ExecuteWithAllowedDirs(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	otherDir := testutil.NewTempDir(t)
	defer otherDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	// Create file in other directory
	otherFile := otherDir.CreateFile("external.txt", "External content")

	// Try to read with allowedDirs
	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":         otherFile,
		"allowed_dirs": []string{otherDir.Path},
	})

	assert.Contains(t, result, "External content")
}

func TestReadTool_ExecuteWithLimit(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	// Create multi-line file with 20 lines
	lines := ""
	for i := 1; i <= 20; i++ {
		lines += fmt.Sprintf("Line %d\n", i)
	}
	testFile := tempDir.CreateFile("large.txt", lines)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":  testFile,
		"limit": float64(10), // JSON解析数字为float64
	})

	// Should contain limit indicator (21 lines total including trailing newline)
	assert.Contains(t, result, "... (11 more lines)")
}

func TestReadTool_ExecuteLargeFile(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	// Create large file (>50KB)
	largeContent := ""
	for i := 0; i < 6000; i++ {
		largeContent += "This is line " + string(rune(i%26+'a')) + " with some content to make it longer\n"
	}
	testFile := tempDir.CreateFile("large.txt", largeContent)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path": testFile,
	})

	// Should be truncated
	assert.LessOrEqual(t, len(result), tool.maxOutput+100)
}

func TestReadTool_Schema(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewReadTool(tempDir.Path)

	schema := tool.Schema()
	assert.Equal(t, "read_file", schema.Function.Name)
	assert.NotNil(t, schema.Function.Parameters)
}

func TestNewWriteTool(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewWriteTool(tempDir.Path)
	assert.NotNil(t, tool)
	assert.Equal(t, "write_file", tool.Name())
}

func TestWriteTool_ExecuteBasic(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewWriteTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "test.txt",
		"content": "Test content",
	})

	assert.Contains(t, result, "Wrote")
	assert.Contains(t, result, "bytes")
	assert.True(t, tempDir.Exists("test.txt"))
	assert.Equal(t, "Test content", tempDir.ReadFile("test.txt"))
}

func TestWriteTool_ExecuteNestedPath(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewWriteTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "internal/engine/test.go",
		"content": "package engine",
	})

	assert.Contains(t, result, "Wrote")
	assert.True(t, tempDir.Exists("internal/engine/test.go"))
}

func TestWriteTool_ExecuteEmptyPath(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewWriteTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "",
		"content": "Test",
	})

	assert.Contains(t, result, "Error: path is required")
}

func TestWriteTool_ExecuteEmptyContent(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewWriteTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "empty.txt",
		"content": "",
	})

	assert.Contains(t, result, "Wrote 0 bytes")
	assert.True(t, tempDir.Exists("empty.txt"))
	assert.Equal(t, "", tempDir.ReadFile("empty.txt"))
}

func TestWriteTool_ExecutePathEscape(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewWriteTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "/etc/hacked",
		"content": "malicious",
	})

	assert.Contains(t, result, "PATH_AUTH_REQUIRED")
}

func TestNewEditTool(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewEditTool(tempDir.Path)
	assert.NotNil(t, tool)
	assert.Equal(t, "edit_file", tool.Name())
}

func TestEditTool_ExecuteBasic(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewEditTool(tempDir.Path)

	// Create initial file
	testFile := tempDir.CreateFile("test.txt", "Hello World")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":     testFile,
		"old_text": "World",
		"new_text": "Go",
	})

	assert.Contains(t, result, "Edited")
	assert.Equal(t, "Hello Go", tempDir.ReadFile("test.txt"))
}

func TestEditTool_ExecuteNotFound(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewEditTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":     "nonexistent.txt",
		"old_text": "old",
		"new_text": "new",
	})

	assert.Contains(t, result, "Error:")
}

func TestEditTool_ExecuteTextNotFound(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewEditTool(tempDir.Path)

	testFile := tempDir.CreateFile("test.txt", "Hello World")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":     testFile,
		"old_text": "nonexistent",
		"new_text": "replacement",
	})

	assert.Contains(t, result, "Error:")
	assert.Contains(t, result, "not found")
}

func TestEditTool_ExecuteMultipleMatches(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewEditTool(tempDir.Path)

	testFile := tempDir.CreateFile("test.txt", "a a a")

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":     testFile,
		"old_text": "a",
		"new_text": "b",
	})

	// Should only replace first occurrence
	assert.Contains(t, result, "Edited")
	assert.Contains(t, tempDir.ReadFile("test.txt"), "b a a")
}

func TestEditTool_ExecuteEmptyPath(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tool := NewEditTool(tempDir.Path)

	result := tool.Execute(context.Background(), map[string]interface{}{
		"path":     "",
		"old_text": "old",
		"new_text": "new",
	})

	assert.Contains(t, result, "Error: path is required")
}
