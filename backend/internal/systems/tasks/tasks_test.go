package tasks

import (
	"encoding/json"
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskManager(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)
	assert.NotNil(t, tm)
	assert.Equal(t, 1, tm.nextID)
}

func TestTaskManager_Create(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	result := tm.Create("Test Task", "Test Description")
	assert.Contains(t, result, "Test Task")
	assert.Contains(t, result, "pending")

	var task TaskRecord
	err := json.Unmarshal([]byte(result), &task)
	require.NoError(t, err)
	assert.Equal(t, 1, task.ID)
	assert.Equal(t, "Test Task", task.Subject)
}

func TestTaskManager_Get(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	tm.Create("Test Task", "Description")
	result := tm.Get(1)
	assert.Contains(t, result, "Test Task")
	assert.NotContains(t, result, "Error")
}

func TestTaskManager_GetNotFound(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	result := tm.Get(999)
	assert.Contains(t, result, "Error")
}

func TestTaskManager_Update(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	tm.Create("Test Task", "Description")
	result := tm.Update(1, "", "", "completed", "test_user", nil, nil)
	assert.Contains(t, result, "completed")
	assert.Contains(t, result, "test_user")
}

func TestTaskManager_UpdateInvalidStatus(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	tm.Create("Test Task", "Description")
	result := tm.Update(1, "", "", "invalid_status", "", nil, nil)
	assert.Contains(t, result, "Error")
	assert.Contains(t, result, "invalid status")
}

func TestTaskManager_ListAll(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	tm.Create("Task 1", "Desc 1")
	tm.Create("Task 2", "Desc 2")

	list := tm.ListAll()
	assert.Contains(t, list, "Task 1")
	assert.Contains(t, list, "Task 2")
}

func TestTaskManager_Dependencies(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	tm.Create("Task 1", "Desc")
	tm.Create("Task 2", "Desc")

	// Task 2 blocks Task 1
	result := tm.Update(2, "", "", "", "", nil, []int{1})
	assert.NotContains(t, result, "Error")

	// Verify dependency on Task 1
	task1Result := tm.Get(1)
	var task1 TaskRecord
	err := json.Unmarshal([]byte(task1Result), &task1)
	require.NoError(t, err)
	assert.Contains(t, task1.BlockedBy, 2)
}

func TestTaskManager_ClearDependencyOnComplete(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tm := NewTaskManager(tempDir.Path)

	tm.Create("Task 1", "Desc")
	tm.Create("Task 2", "Desc")

	// Task 2 blocks Task 1
	tm.Update(2, "", "", "", "", nil, []int{1})

	// Complete Task 2 - should clear dependency
	tm.Update(2, "", "", "completed", "", nil, nil)

	// Verify Task 1 no longer blocked
	task1Result := tm.Get(1)
	var task1 TaskRecord
	err := json.Unmarshal([]byte(task1Result), &task1)
	require.NoError(t, err)
	assert.NotContains(t, task1.BlockedBy, 2)
}
