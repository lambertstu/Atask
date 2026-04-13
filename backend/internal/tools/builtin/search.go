package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

type SearchFilesTool struct {
	workDir   string
	maxOutput int
}

func NewSearchFilesTool(workDir string) tools.Tool {
	return &SearchFilesTool{
		workDir:   workDir,
		maxOutput: 50000,
	}
}

func (t *SearchFilesTool) Name() string {
	return "search_files"
}

func (t *SearchFilesTool) Description() string {
	return "Search for files by name matching a pattern. Use relative path for dir."
}

func (t *SearchFilesTool) Execute(ctx context.Context, args map[string]interface{}) string {
	pattern := utils.GetStringFromMap(args, "pattern")
	if pattern == "" {
		return "Error: pattern is required"
	}
	dir := utils.GetStringFromMap(args, "dir")
	if dir == "" {
		dir = "."
	}

	allowedDirs := utils.GetStringSliceFromMap(args, "allowed_dirs")
	safe, err := utils.SafePath(t.workDir, allowedDirs, dir)
	if err != nil {
		if _, ok := err.(*utils.PathEscapeError); ok {
			return fmt.Sprintf("PATH_AUTH_REQUIRED:%s", safe)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	if !strings.Contains(pattern, "*") {
		pattern = "*" + pattern + "*"
	}

	var results []string
	err = filepath.WalkDir(safe, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".cursor" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		matched, _ := filepath.Match(pattern, d.Name())
		if matched {
			rel, _ := filepath.Rel(t.workDir, path)
			results = append(results, rel)
		}
		return nil
	})

	if err != nil {
		return fmt.Sprintf("Error searching files: %v", err)
	}

	if len(results) == 0 {
		return "No files found"
	}

	out := strings.Join(results, "\n")
	if len(out) > t.maxOutput {
		out = out[:t.maxOutput] + "\n... (truncated)"
	}
	return out
}

func (t *SearchFilesTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "File name pattern (e.g. *skill*)",
					},
					"dir": map[string]interface{}{
						"type":        "string",
						"description": "Relative directory to search in, defaults to workspace root",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

type GrepCodeTool struct {
	workDir   string
	maxOutput int
}

func NewGrepCodeTool(workDir string) tools.Tool {
	return &GrepCodeTool{
		workDir:   workDir,
		maxOutput: 50000,
	}
}

func (t *GrepCodeTool) Name() string {
	return "grep_code"
}

func (t *GrepCodeTool) Description() string {
	return "Search for text or regex patterns within file contents. Use relative path for dir."
}

func (t *GrepCodeTool) Execute(ctx context.Context, args map[string]interface{}) string {
	pattern := utils.GetStringFromMap(args, "pattern")
	if pattern == "" {
		return "Error: pattern is required"
	}
	dir := utils.GetStringFromMap(args, "dir")
	if dir == "" {
		dir = "."
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Sprintf("Error: invalid regex pattern: %v", err)
	}

	allowedDirs := utils.GetStringSliceFromMap(args, "allowed_dirs")
	safe, err := utils.SafePath(t.workDir, allowedDirs, dir)
	if err != nil {
		if _, ok := err.(*utils.PathEscapeError); ok {
			return fmt.Sprintf("PATH_AUTH_REQUIRED:%s", safe)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	var results []string
	matchesCount := 0

	err = filepath.WalkDir(safe, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == ".cursor" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil || info.Size() > 1024*1024 {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		rel, _ := filepath.Rel(t.workDir, path)
		scanner := bufio.NewScanner(file)
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d:%s", rel, lineNum, strings.TrimSpace(line)))
				matchesCount++
				if matchesCount > 500 {
					return fmt.Errorf("too many matches")
				}
			}
			lineNum++
		}
		return nil
	})

	if err != nil && err.Error() != "too many matches" {
		return fmt.Sprintf("Error searching files: %v", err)
	}

	if len(results) == 0 {
		return "No matches found"
	}

	out := strings.Join(results, "\n")
	if err != nil && err.Error() == "too many matches" {
		out += "\n... (truncated due to too many matches)"
	}

	if len(out) > t.maxOutput {
		out = out[:t.maxOutput] + "\n... (truncated)"
	}
	return out
}

func (t *GrepCodeTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Regex pattern to search for",
					},
					"dir": map[string]interface{}{
						"type":        "string",
						"description": "Relative directory to search in, defaults to workspace root",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}
