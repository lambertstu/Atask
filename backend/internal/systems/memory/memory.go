package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

const (
	MaxIndexLines = 200
)

var MemoryTypes = []string{"user", "feedback", "project", "reference"}

type Memory struct {
	Name        string
	Description string
	Type        string
	Content     string
	File        string
}

type MemoryManager struct {
	memoryDir string
	memories  map[string]Memory
}

func NewMemoryManager(memoryDir string) *MemoryManager {
	return &MemoryManager{
		memoryDir: memoryDir,
		memories:  make(map[string]Memory),
	}
}

func (m *MemoryManager) LoadAll() {
	if _, err := os.Stat(m.memoryDir); os.IsNotExist(err) {
		return
	}

	filepath.Walk(m.memoryDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".md") || info.Name() == "MEMORY.md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		name, memType, desc, body := parseMemoryFile(string(content))
		if name == "" {
			name = strings.TrimSuffix(info.Name(), ".md")
		}

		m.memories[name] = Memory{
			Name:        name,
			Description: desc,
			Type:        memType,
			Content:     body,
			File:        info.Name(),
		}
		return nil
	})

	if len(m.memories) > 0 {
		fmt.Printf("[Memory loaded: %d memories from %s]\n", len(m.memories), m.memoryDir)
	}
}

func (m *MemoryManager) ListMemories() map[string]Memory {
	return m.memories
}

func (m *MemoryManager) LoadMemoryPrompt() string {
	if len(m.memories) == 0 {
		return ""
	}

	var sections []string
	sections = append(sections, "# Memories (persistent across sessions)")
	sections = append(sections, "")

	for _, memType := range MemoryTypes {
		var typed []Memory
		for _, mem := range m.memories {
			if mem.Type == memType {
				typed = append(typed, mem)
			}
		}
		if len(typed) == 0 {
			continue
		}

		sections = append(sections, fmt.Sprintf("## [%s]", memType))
		for _, mem := range typed {
			sections = append(sections, fmt.Sprintf("### %s: %s", mem.Name, mem.Description))
			if mem.Content != "" {
				sections = append(sections, mem.Content)
			}
			sections = append(sections, "")
		}
	}

	return strings.Join(sections, "\n")
}

func (m *MemoryManager) SaveMemory(name, description, memType, content string) string {
	if !contains(MemoryTypes, memType) {
		return fmt.Sprintf("Error: type must be one of %v", MemoryTypes)
	}

	safeName := regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(strings.ToLower(name), "_")
	if safeName == "" {
		return "Error: invalid memory name"
	}

	os.MkdirAll(m.memoryDir, 0755)

	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\ntype: %s\n---\n%s\n", name, description, memType, content)
	fileName := safeName + ".md"
	filePath := filepath.Join(m.memoryDir, fileName)

	if err := os.WriteFile(filePath, []byte(frontmatter), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	m.memories[name] = Memory{
		Name:        name,
		Description: description,
		Type:        memType,
		Content:     content,
		File:        fileName,
	}

	m.rebuildIndex()

	return fmt.Sprintf("Saved memory '%s' [%s] to %s", name, memType, filePath)
}

func (m *MemoryManager) rebuildIndex() {
	var lines []string
	lines = append(lines, "# Memory Index", "")

	var names []string
	for name := range m.memories {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		mem := m.memories[name]
		lines = append(lines, fmt.Sprintf("- %s: %s [%s]", name, mem.Description, mem.Type))
		if len(lines) >= MaxIndexLines {
			lines = append(lines, fmt.Sprintf("... (truncated at %d lines)", MaxIndexLines))
			break
		}
	}

	os.MkdirAll(m.memoryDir, 0755)
	indexPath := filepath.Join(m.memoryDir, "MEMORY.md")
	os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func parseMemoryFile(text string) (name, memType, description, body string) {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return "", "", "", strings.TrimSpace(text)
	}

	header := match[1]
	body = strings.TrimSpace(match[2])
	memType = "project"

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

	return name, memType, description, body
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type SaveMemoryTool struct {
	manager *MemoryManager
}

func NewSaveMemoryTool(manager *MemoryManager) tools.Tool {
	return &SaveMemoryTool{manager: manager}
}

func (t *SaveMemoryTool) Name() string {
	return "save_memory"
}

func (t *SaveMemoryTool) Description() string {
	return "Save a memory for persistent knowledge across sessions."
}

func (t *SaveMemoryTool) Execute(ctx context.Context, args map[string]interface{}) string {
	name := utils.GetStringFromMap(args, "name")
	description := utils.GetStringFromMap(args, "description")
	memType := utils.GetStringFromMap(args, "type")
	content := utils.GetStringFromMap(args, "content")

	if name == "" {
		return "Error: name is required"
	}
	return t.manager.SaveMemory(name, description, memType, content)
}

func (t *SaveMemoryTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Memory name",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Brief description",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Memory type: user, feedback, project, reference",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Memory content",
					},
				},
				"required": []string{"name"},
			},
		},
	}
}
