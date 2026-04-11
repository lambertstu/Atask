package tasks

import (
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewCronScheduler(t *testing.T) {
	cs := NewCronScheduler()
	assert.NotNil(t, cs)
}

func TestCronScheduler_Create(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	cs := NewCronSchedulerWithWorkDir(tempDir.Path)

	taskID := cs.Create("* * * * *", "test prompt", false, false)
	assert.NotEmpty(t, taskID)
}

func TestCronScheduler_Delete(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	cs := NewCronSchedulerWithWorkDir(tempDir.Path)

	taskID := cs.Create("* * * * *", "test", false, false)

	result := cs.Delete(taskID)
	assert.NotContains(t, result, "Error")
}

func TestCronScheduler_ListTasks(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	cs := NewCronSchedulerWithWorkDir(tempDir.Path)

	cs.Create("* * * * *", "task1", false, false)
	cs.Create("0 * * * *", "task2", true, false)

	list := cs.ListTasks()
	assert.Contains(t, list, "task1")
	assert.Contains(t, list, "task2")
}

func TestCronScheduler_StartStop(t *testing.T) {
	cs := NewCronScheduler()

	cs.Start()
	cs.Stop()
}
