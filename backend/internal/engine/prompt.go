package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DYNAMIC_BOUNDARY = "=== DYNAMIC_BOUNDARY ==="

type SystemPromptBuilder struct {
	workdir    string
	projectDir string
}

func NewSystemPromptBuilder(workdir, projectDir string) *SystemPromptBuilder {
	return &SystemPromptBuilder{
		workdir:    workdir,
		projectDir: projectDir,
	}
}

func (b *SystemPromptBuilder) Build() string {
	var sections []string

	if core := b.buildCore(); core != "" {
		sections = append(sections, core)
	}

	if tools := b.buildToolListing(); tools != "" {
		sections = append(sections, tools)
	}

	if skills := b.buildSkillListing(); skills != "" {
		sections = append(sections, skills)
	}

	if memory := b.buildMemorySection(); memory != "" {
		sections = append(sections, memory)
	}

	if claudeMd := b.buildClaudeMD(); claudeMd != "" {
		sections = append(sections, claudeMd)
	}

	sections = append(sections, DYNAMIC_BOUNDARY)

	if dynamic := b.buildDynamicContext(); dynamic != "" {
		sections = append(sections, dynamic)
	}

	return strings.Join(sections, "\n\n")
}

func (b *SystemPromptBuilder) buildCore() string {
	return fmt.Sprintf("You are a coding agent operating in %s.\nUse the provided tools to explore, read, write, and edit files.\nAlways verify before assuming. Prefer reading files over guessing.", b.workdir)
}

func (b *SystemPromptBuilder) buildToolListing() string {
	return ""
}

func (b *SystemPromptBuilder) buildSkillListing() string {
	skillsDir := filepath.Join(b.projectDir, "skills")
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return ""
	}

	var skills []string
	filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		name, desc := parseSkillFrontmatter(string(content))
		if name == "" {
			name = filepath.Base(filepath.Dir(path))
		}
		skills = append(skills, fmt.Sprintf("- %s: %s", name, desc))
		return nil
	})

	if len(skills) == 0 {
		return ""
	}
	return "# Available skills\n" + strings.Join(skills, "\n")
}

func (b *SystemPromptBuilder) buildMemorySection() string {
	memoryDir := filepath.Join(b.workdir, ".memory")
	if _, err := os.Stat(memoryDir); os.IsNotExist(err) {
		return ""
	}

	var memories []string
	filepath.Walk(memoryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") || info.Name() == "MEMORY.md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		name, memType, desc, body := parseMemoryFrontmatter(string(content))
		if name == "" {
			name = strings.TrimSuffix(info.Name(), ".md")
		}
		memories = append(memories, fmt.Sprintf("[%s] %s: %s\n%s", memType, name, desc, body))
		return nil
	})

	if len(memories) == 0 {
		return ""
	}
	return "# Memories (persistent)\n\n" + strings.Join(memories, "\n\n")
}

func (b *SystemPromptBuilder) buildClaudeMD() string {
	var sources []string

	userClaude := filepath.Join(os.Getenv("HOME"), ".claude", "CLAUDE.md")
	if content, err := os.ReadFile(userClaude); err == nil {
		sources = append(sources, fmt.Sprintf("## From user global (~/.claude/CLAUDE.md)\n%s", string(content)))
	}

	projectClaude := filepath.Join(b.workdir, "CLAUDE.md")
	if content, err := os.ReadFile(projectClaude); err == nil {
		sources = append(sources, fmt.Sprintf("## From project root (CLAUDE.md)\n%s", string(content)))
	}

	cwd, _ := os.Getwd()
	if cwd != b.workdir {
		subdirClaude := filepath.Join(cwd, "CLAUDE.md")
		if content, err := os.ReadFile(subdirClaude); err == nil {
			sources = append(sources, fmt.Sprintf("## From subdir (%s/CLAUDE.md)\n%s", filepath.Base(cwd), string(content)))
		}
	}

	if len(sources) == 0 {
		return ""
	}
	return "# CLAUDE.md instructions\n\n" + strings.Join(sources, "\n\n")
}

func (b *SystemPromptBuilder) buildDynamicContext() string {
	hostname, _ := os.Hostname()
	lines := []string{
		fmt.Sprintf("Current date: %s", time.Now().Format("2006-01-02")),
		fmt.Sprintf("Working directory: %s", b.workdir),
		fmt.Sprintf("Platform: %s", hostname),
	}
	return "# Dynamic context\n" + strings.Join(lines, "\n")
}

func parseSkillFrontmatter(text string) (name, description string) {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return "", ""
	}

	header := match[1]
	for _, line := range strings.Split(header, "\n") {
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return name, description
}

func parseMemoryFrontmatter(text string) (name, memType, description, body string) {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return "", "", "", ""
	}

	header := match[1]
	body = strings.TrimSpace(match[2])

	for _, line := range strings.Split(header, "\n") {
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "type:") {
			memType = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}

	if memType == "" {
		memType = "project"
	}
	return name, memType, description, body
}
