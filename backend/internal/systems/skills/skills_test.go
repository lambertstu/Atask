package skills

import (
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewSkillLoader(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	sl := NewSkillLoader(tempDir.Path)
	assert.NotNil(t, sl)
}

func TestSkillLoader_LoadAll(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	// Create a skill directory and SKILL.md
	skillDir := tempDir.Subdir("test-skill")
	skillContent := `---
name: test-skill
description: Test skill
---
# Test Skill Content`
	skillDir.CreateFile("SKILL.md", skillContent)

	sl := NewSkillLoader(tempDir.Path)
	sl.LoadAll()

	descs := sl.GetDescriptions()
	assert.Contains(t, descs, "test-skill")
}

func TestSkillLoader_GetDescriptions(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	sl := NewSkillLoader(tempDir.Path)
	sl.LoadAll()

	descs := sl.GetDescriptions()
	assert.NotNil(t, descs)
}

func TestSkillLoader_GetContent(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	sl := NewSkillLoader(tempDir.Path)

	content := sl.GetContent("nonexistent")
	// Returns error message about available skills
	assert.NotEmpty(t, content)
}
