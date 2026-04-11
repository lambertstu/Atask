package tasks

import (
	"testing"
	"time"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewBackgroundManager(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	bm := NewBackgroundManager(tempDir.Path)
	assert.NotNil(t, bm)
}

func TestBackgroundManager_Run(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	bm := NewBackgroundManagerWithWorkDir(tempDir.Path, tempDir.Path)

	taskID := bm.Run("echo hello")
	assert.NotEmpty(t, taskID)
}

func TestBackgroundManager_Check(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	bm := NewBackgroundManagerWithWorkDir(tempDir.Path, tempDir.Path)

	taskID := bm.Run("echo test")

	// Wait for completion
	time.Sleep(200 * time.Millisecond)

	result := bm.Check(taskID)
	// Result should contain status info
	assert.NotEmpty(t, result)
}

func TestBackgroundManager_DrainNotifications(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	bm := NewBackgroundManagerWithWorkDir(tempDir.Path, tempDir.Path)

	notifications := bm.DrainNotifications()
	assert.NotNil(t, notifications)
}
