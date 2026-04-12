package builtin

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

type BashTool struct {
	workDir   string
	timeout   time.Duration
	maxOutput int
	dangerous []string
}

func NewBashTool(workDir string, timeoutSeconds int) *BashTool {
	return &BashTool{
		workDir:   workDir,
		timeout:   time.Duration(timeoutSeconds) * time.Second,
		maxOutput: 50000,
		dangerous: []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"},
	}
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return "Run a shell command."
}

func (t *BashTool) Execute(ctx context.Context, args map[string]interface{}) string {
	command := utils.GetStringFromMap(args, "command")
	if command == "" {
		return "Error: command is required"
	}

	for _, d := range t.dangerous {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}

	workDir := t.workDir
	if sessionWorkDir, ok := args["_session_work_dir"].(string); ok && sessionWorkDir != "" {
		workDir = sessionWorkDir
	}

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "Error: Timeout (120s)"
	}

	out := strings.TrimSpace(string(output))
	if err != nil && out == "" {
		return fmt.Sprintf("Error: %v", err)
	}

	if len(out) == 0 {
		return "(no output)"
	}

	if len(out) > t.maxOutput {
		out = out[:t.maxOutput]
	}

	return out
}

func (t *BashTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func RegisterBashTool(registry tools.ToolRegistry, workDir string, timeoutSeconds int) {
	registry.Register(NewBashTool(workDir, timeoutSeconds))
}
