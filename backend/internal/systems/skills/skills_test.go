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

func TestSkillLoader_LoadAll_FolderNameFallback(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	// Create a skill directory and SKILL.md without name in frontmatter
	skillDir := tempDir.Subdir("fallback-skill")
	skillContent := `---
description: Test skill fallback
---
# Test Skill Content Fallback`
	skillDir.CreateFile("SKILL.md", skillContent)

	sl := NewSkillLoader(tempDir.Path)
	sl.LoadAll()

	descs := sl.GetDescriptions()
	assert.Contains(t, descs, "fallback-skill")
}

func TestSkillLoader_LoadAll_Conflict(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	// Create first skill
	skillDir1 := tempDir.Subdir("conflict-skill-1")
	skillContent1 := `---
name: same-name
description: First skill
---
# First Content`
	skillDir1.CreateFile("SKILL.md", skillContent1)

	// Create second skill with the SAME name
	skillDir2 := tempDir.Subdir("conflict-skill-2")
	skillContent2 := `---
name: same-name
description: Second skill
---
# Second Content`
	skillDir2.CreateFile("SKILL.md", skillContent2)

	sl := NewSkillLoader(tempDir.Path)
	sl.LoadAll()

	// Should only load one skill with the name "same-name"
	assert.Equal(t, 1, len(sl.skills))
	assert.Contains(t, sl.skills, "same-name")

	// Either one could be loaded depending on filepath.Walk order, but not both or panicked
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
