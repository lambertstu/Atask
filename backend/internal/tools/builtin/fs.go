package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

type ReadTool struct {
	workDir   string
	maxOutput int
}

type WriteTool struct {
	workDir string
}

type EditTool struct {
	workDir string
}

func NewReadTool(workDir string) *ReadTool {
	return &ReadTool{
		workDir:   workDir,
		maxOutput: 50000,
	}
}

func NewWriteTool(workDir string) *WriteTool {
	return &WriteTool{workDir: workDir}
}

func NewEditTool(workDir string) *EditTool {
	return &EditTool{workDir: workDir}
}

func NewReadFileTool(workDir string) tools.Tool {
	return NewReadTool(workDir)
}

func NewWriteFileTool(workDir string) tools.Tool {
	return NewWriteTool(workDir)
}

func NewEditFileTool(workDir string) tools.Tool {
	return NewEditTool(workDir)
}

func (t *ReadTool) Name() string {
	return "read_file"
}

func (t *ReadTool) Description() string {
	return "Read file contents. Use relative path from workspace root."
}

func (t *ReadTool) Execute(ctx context.Context, args map[string]interface{}) string {
	path := utils.GetStringFromMap(args, "path")
	if path == "" {
		return "Error: path is required"
	}

	limit := utils.GetIntFromMap(args, "limit")

	workDir := t.workDir
	if sessionWorkDir, ok := args["_session_work_dir"].(string); ok && sessionWorkDir != "" {
		workDir = sessionWorkDir
	}

	allowedDirs := utils.GetStringSliceFromMap(args, "allowed_dirs")
	safe, err := utils.SafePath(workDir, allowedDirs, path)
	if err != nil {
		if _, ok := err.(*utils.PathEscapeError); ok {
			return fmt.Sprintf("PATH_AUTH_REQUIRED:%s", safe)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	content, err := os.ReadFile(safe)
	if err != nil {
		if os.IsNotExist(err) {
			dir := filepath.Dir(safe)
			base := filepath.Base(safe)
			entries, dirErr := os.ReadDir(dir)
			if dirErr == nil {
				var suggestions []string
				for _, e := range entries {
					if strings.Contains(e.Name(), base) || strings.Contains(base, e.Name()) {
						suggestions = append(suggestions, e.Name())
					}
				}
				if len(suggestions) > 0 {
					return fmt.Sprintf("Error: file not found: %s. Did you mean: %s?", path, strings.Join(suggestions, ", "))
				}
			}
			return fmt.Sprintf("Error: file not found: %s", path)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	if limit > 0 && limit < len(lines) {
		lines = append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit))
	}

	out := strings.Join(lines, "\n")
	if len(out) > t.maxOutput {
		out = out[:t.maxOutput]
	}

	return out
}

func (t *ReadTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative path from workspace root",
					},
					"limit": map[string]interface{}{
						"type": "integer",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *WriteTool) Name() string {
	return "write_file"
}

func (t *WriteTool) Description() string {
	return "Write content to file. Use relative path from workspace root."
}

func (t *WriteTool) Execute(ctx context.Context, args map[string]interface{}) string {
	path := utils.GetStringFromMap(args, "path")
	content := utils.GetStringFromMap(args, "content")

	if path == "" {
		return "Error: path is required"
	}

	workDir := t.workDir
	if sessionWorkDir, ok := args["_session_work_dir"].(string); ok && sessionWorkDir != "" {
		workDir = sessionWorkDir
	}

	allowedDirs := utils.GetStringSliceFromMap(args, "allowed_dirs")
	safe, err := utils.SafePath(workDir, allowedDirs, path)
	if err != nil {
		if _, ok := err.(*utils.PathEscapeError); ok {
			return fmt.Sprintf("PATH_AUTH_REQUIRED:%s", safe)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(safe), 0755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if err := os.WriteFile(safe, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}

func (t *WriteTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative path from workspace root",
					},
					"content": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *EditTool) Name() string {
	return "edit_file"
}

func (t *EditTool) Description() string {
	return "Replace exact text in file. Use relative path from workspace root."
}

func (t *EditTool) Execute(ctx context.Context, args map[string]interface{}) string {
	path := utils.GetStringFromMap(args, "path")
	oldText := utils.GetStringFromMap(args, "old_text")
	newText := utils.GetStringFromMap(args, "new_text")

	if path == "" {
		return "Error: path is required"
	}

	workDir := t.workDir
	if sessionWorkDir, ok := args["_session_work_dir"].(string); ok && sessionWorkDir != "" {
		workDir = sessionWorkDir
	}

	allowedDirs := utils.GetStringSliceFromMap(args, "allowed_dirs")
	safe, err := utils.SafePath(workDir, allowedDirs, path)
	if err != nil {
		if _, ok := err.(*utils.PathEscapeError); ok {
			return fmt.Sprintf("PATH_AUTH_REQUIRED:%s", safe)
		}
		return fmt.Sprintf("Error: %v", err)
	}

	content, err := os.ReadFile(safe)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, oldText) {
		return fmt.Sprintf("Error: Text not found in %s", path)
	}

	newContent := strings.Replace(strContent, oldText, newText, 1)
	if err := os.WriteFile(safe, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return fmt.Sprintf("Edited %s", path)
}

func (t *EditTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Relative path from workspace root",
					},
					"old_text": map[string]interface{}{
						"type": "string",
					},
					"new_text": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"path", "old_text", "new_text"},
			},
		},
	}
}

func RegisterFSTools(registry tools.ToolRegistry, workDir string) {
	registry.Register(NewReadTool(workDir))
	registry.Register(NewWriteTool(workDir))
	registry.Register(NewEditTool(workDir))
}
