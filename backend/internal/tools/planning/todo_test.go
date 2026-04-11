package planning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTodoManager(t *testing.T) {
	manager := NewTodoManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 0, len(manager.items))
}

func TestTodoManager_UpdateBasic(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		map[string]interface{}{"id": "1", "text": "Task 1", "status": "pending"},
		map[string]interface{}{"id": "2", "text": "Task 2", "status": "pending"},
	}

	result, err := manager.Update(items)
	require.NoError(t, err)
	assert.Contains(t, result, "[ ]")
	assert.Contains(t, result, "Task 1")
	assert.Contains(t, result, "Task 2")
}

func TestTodoManager_UpdateWithProcessing(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		map[string]interface{}{"id": "1", "text": "Task 1", "status": "processing"},
		map[string]interface{}{"id": "2", "text": "Task 2", "status": "pending"},
	}

	result, err := manager.Update(items)
	require.NoError(t, err)
	assert.Contains(t, result, "[>]")
	assert.Contains(t, result, "Task 1")
}

func TestTodoManager_UpdateMultipleProcessing(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		map[string]interface{}{"id": "1", "text": "Task 1", "status": "processing"},
		map[string]interface{}{"id": "2", "text": "Task 2", "status": "processing"},
	}

	result, err := manager.Update(items)
	require.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "only one task processing")
}

func TestTodoManager_UpdateWithComplete(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		map[string]interface{}{"id": "1", "text": "Task 1", "status": "complete"},
		map[string]interface{}{"id": "2", "text": "Task 2", "status": "pending"},
	}

	result, err := manager.Update(items)
	require.NoError(t, err)
	assert.Contains(t, result, "[√]")
	assert.Contains(t, result, "(1/2 complete)")
}

func TestTodoManager_UpdateInvalidInput(t *testing.T) {
	manager := NewTodoManager()

	result, err := manager.Update("invalid")
	require.Error(t, err)
	assert.Equal(t, "", result)
	assert.Contains(t, err.Error(), "must be an array")
}

func TestTodoManager_UpdateEmptyStatus(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		map[string]interface{}{"id": "1", "text": "Task 1"},
	}

	result, err := manager.Update(items)
	require.NoError(t, err)
	assert.Contains(t, result, "[ ]")
}

func TestTodoManager_UpdateInvalidItem(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		"invalid item",
		map[string]interface{}{"id": "2", "text": "Task 2", "status": "pending"},
	}

	result, err := manager.Update(items)
	require.NoError(t, err)
	assert.Contains(t, result, "Task 2")
}

func TestTodoManager_RenderEmpty(t *testing.T) {
	manager := NewTodoManager()

	result := manager.Render()
	assert.Equal(t, "no todo list", result)
}

func TestTodoManager_RenderAllComplete(t *testing.T) {
	manager := NewTodoManager()

	items := []interface{}{
		map[string]interface{}{"id": "1", "text": "Task 1", "status": "complete"},
		map[string]interface{}{"id": "2", "text": "Task 2", "status": "complete"},
	}

	manager.Update(items)
	result := manager.Render()

	assert.Contains(t, result, "[√]")
	assert.Contains(t, result, "(2/2 complete)")
}

func TestNewTodoTool(t *testing.T) {
	tool := NewTodoTool()
	assert.NotNil(t, tool)
	assert.Equal(t, "todo", tool.Name())
	assert.Equal(t, "Update task list. Track progress on multi-step tasks.", tool.Description())
}

func TestTodoTool_Execute(t *testing.T) {
	tool := NewTodoTool()

	args := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"id": "1", "text": "Task 1", "status": "pending"},
		},
	}

	result := tool.Execute(context.Background(), args)
	assert.Contains(t, result, "[ ]")
	assert.Contains(t, result, "Task 1")
}

func TestTodoTool_ExecuteError(t *testing.T) {
	tool := NewTodoTool()

	args := map[string]interface{}{
		"items": "invalid",
	}

	result := tool.Execute(context.Background(), args)
	assert.Contains(t, result, "must be an array")
}

func TestTodoTool_Schema(t *testing.T) {
	tool := NewTodoTool()

	schema := tool.Schema()
	assert.Equal(t, "todo", schema.Function.Name)
	assert.NotNil(t, schema.Function.Parameters)
}
