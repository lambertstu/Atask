package skills

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
	"gopkg.in/yaml.v3"
)

type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Tags        string `yaml:"tags"`
}

type Skill struct {
	Meta SkillMeta
	Body string
	Path string
}

type SkillLoader struct {
	skillsDir string
	skills    map[string]Skill
}

func NewSkillLoader(skillsDir string) *SkillLoader {
	return &SkillLoader{
		skillsDir: skillsDir,
		skills:    make(map[string]Skill),
	}
}

func (l *SkillLoader) LoadAll() {
	if _, err := os.Stat(l.skillsDir); os.IsNotExist(err) {
		return
	}

	filepath.Walk(l.skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		meta, body := l.parseFrontmatter(string(content))
		if meta.Name == "" {
			meta.Name = filepath.Base(filepath.Dir(path))
		}

		if existing, ok := l.skills[meta.Name]; ok {
			fmt.Printf("\033[33m[Warning]\033[0m Skill name conflict detected: '%s'. Skipping %s (already loaded from %s)\n",
				meta.Name, path, existing.Path)
			return nil
		}

		l.skills[meta.Name] = Skill{
			Meta: meta,
			Body: body,
			Path: path,
		}
		return nil
	})
}

func (l *SkillLoader) parseFrontmatter(text string) (SkillMeta, string) {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)`)
	match := re.FindStringSubmatch(text)
	if match == nil {
		return SkillMeta{}, strings.TrimSpace(text)
	}

	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(match[1]), &meta); err != nil {
		for _, line := range strings.Split(match[1], "\n") {
			if strings.HasPrefix(line, "name:") {
				meta.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			}
			if strings.HasPrefix(line, "description:") {
				meta.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			}
		}
	}

	return meta, strings.TrimSpace(match[2])
}

func (l *SkillLoader) GetDescriptions() string {
	if len(l.skills) == 0 {
		return "(no skills available)"
	}

	var names []string
	for name := range l.skills {
		names = append(names, name)
	}
	sort.Strings(names)

	var lines []string
	for _, name := range names {
		skill := l.skills[name]
		line := fmt.Sprintf("  - %s: %s", name, skill.Meta.Description)
		if skill.Meta.Tags != "" {
			line += fmt.Sprintf(" [%s]", skill.Meta.Tags)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (l *SkillLoader) GetContent(name string) string {
	skill, ok := l.skills[name]
	if !ok {
		available := make([]string, 0, len(l.skills))
		for n := range l.skills {
			available = append(available, n)
		}
		return fmt.Sprintf("Error: Unknown skill '%s'. Available: %s", name, strings.Join(available, ", "))
	}
	return fmt.Sprintf("<skill name=\"%s\">\n%s\n</skill>", name, skill.Body)
}

type LoadSkillTool struct {
	loader *SkillLoader
}

func NewLoadSkillTool(loader *SkillLoader) tools.Tool {
	return &LoadSkillTool{loader: loader}
}

func (t *LoadSkillTool) Name() string {
	return "load_skill"
}

func (t *LoadSkillTool) Description() string {
	return "Load a skill knowledge file. Returns skill content."
}

func (t *LoadSkillTool) Execute(ctx context.Context, args map[string]interface{}) string {
	name := utils.GetStringFromMap(args, "name")
	if name == "" {
		return "Error: name is required"
	}
	return t.loader.GetContent(name)
}

func (t *LoadSkillTool) Schema() openai.Tool {
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
						"description": "Skill name to load",
					},
				},
				"required": []string{"name"},
			},
		},
	}
}
