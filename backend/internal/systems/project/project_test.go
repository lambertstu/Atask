package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectManager_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	pm := NewProjectManager(tempDir)

	projectPath := "/Users/libing/gitProject/collect-crawler"
	project := pm.GetOrCreate(projectPath)

	assert.Equal(t, projectPath, project.Path)
	assert.NotNil(t, project.Sessions)
	assert.Equal(t, 0, len(project.Sessions))

	pm.AddSession(projectPath, "session-1")
	pm.AddSession(projectPath, "session-2")
	pm.AddSession(projectPath, "session-1") // duplicate

	project = pm.GetProject(projectPath)
	assert.Equal(t, 2, len(project.Sessions))
	assert.Contains(t, project.Sessions, "session-1")
	assert.Contains(t, project.Sessions, "session-2")

	projects := pm.ListProjects()
	assert.Equal(t, 1, len(projects))

	// Test persistence
	pm2 := NewProjectManager(tempDir)
	projects2 := pm2.ListProjects()
	assert.Equal(t, 1, len(projects2))

	project2 := pm2.GetProject(projectPath)
	assert.NotNil(t, project2)
	assert.Equal(t, 2, len(project2.Sessions))
}
