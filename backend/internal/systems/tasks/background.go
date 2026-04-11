package tasks

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

const StallThresholdS = 45

type BackgroundTask struct {
	ID            string  `json:"id"`
	Status        string  `json:"status"`
	Result        string  `json:"result,omitempty"`
	Command       string  `json:"command"`
	StartedAt     float64 `json:"started_at"`
	FinishedAt    float64 `json:"finished_at,omitempty"`
	ResultPreview string  `json:"result_preview"`
	OutputFile    string  `json:"output_file"`
}

type BackgroundNotification struct {
	TaskID     string
	Status     string
	Command    string
	Preview    string
	OutputFile string
}

type BackgroundManager struct {
	dir               string
	workDir           string
	tasks             map[string]*BackgroundTask
	notificationQueue []BackgroundNotification
	mu                sync.Mutex
}

func NewBackgroundManager(dir string) *BackgroundManager {
	workDir := filepath.Dir(dir)
	return &BackgroundManager{
		dir:               dir,
		workDir:           workDir,
		tasks:             make(map[string]*BackgroundTask),
		notificationQueue: []BackgroundNotification{},
	}
}

func NewBackgroundManagerWithWorkDir(dir, workDir string) *BackgroundManager {
	return &BackgroundManager{
		dir:               dir,
		workDir:           workDir,
		tasks:             make(map[string]*BackgroundTask),
		notificationQueue: []BackgroundNotification{},
	}
}

func (bm *BackgroundManager) recordPath(taskID string) string {
	return filepath.Join(bm.dir, taskID+".json")
}

func (bm *BackgroundManager) outputPath(taskID string) string {
	return filepath.Join(bm.dir, taskID+".log")
}

func (bm *BackgroundManager) persistTask(taskID string) {
	task := bm.tasks[taskID]
	data, _ := json.MarshalIndent(task, "", "  ")
	os.WriteFile(bm.recordPath(taskID), data, 0644)
}

func (bm *BackgroundManager) preview(output string, limit int) string {
	compact := strings.Join(strings.Fields(output), " ")
	if len(compact) > limit {
		compact = compact[:limit]
	}
	if compact == "" {
		compact = "(no output)"
	}
	return compact
}

func (bm *BackgroundManager) Run(command string) string {
	os.MkdirAll(bm.dir, 0755)

	taskID := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)[:8]
	outputFile := bm.outputPath(taskID)

	task := &BackgroundTask{
		ID:         taskID,
		Status:     "running",
		Command:    command,
		StartedAt:  float64(time.Now().Unix()),
		OutputFile: outputFile,
	}
	bm.tasks[taskID] = task
	bm.persistTask(taskID)

	go bm.execute(taskID, command)

	relOutputFile, _ := filepath.Rel(bm.workDir, outputFile)
	return fmt.Sprintf("Background task %s started: %s (output_file=%s)", taskID, utils.Truncate(command, 80), relOutputFile)
}

func (bm *BackgroundManager) execute(taskID string, command string) {
	ctx, cancel := gocontext.WithTimeout(gocontext.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = bm.workDir

	output, err := cmd.CombinedOutput()
	var status string
	var finalOutput string

	if ctx.Err() == gocontext.DeadlineExceeded {
		status = "timeout"
		finalOutput = "Error: Timeout (300s)"
	} else if err != nil {
		status = "error"
		finalOutput = fmt.Sprintf("Error: %v", err)
	} else {
		status = "completed"
		finalOutput = string(output)
	}

	if len(finalOutput) > 50000 {
		finalOutput = finalOutput[:50000]
	}
	if finalOutput == "" {
		finalOutput = "(no output)"
	}

	preview := bm.preview(finalOutput, 500)
	outputPath := bm.outputPath(taskID)
	os.WriteFile(outputPath, []byte(finalOutput), 0644)

	bm.mu.Lock()
	if task, ok := bm.tasks[taskID]; ok {
		task.Status = status
		task.Result = finalOutput
		task.FinishedAt = float64(time.Now().Unix())
		task.ResultPreview = preview
		bm.persistTask(taskID)

		relOutputFile, _ := filepath.Rel(bm.workDir, outputPath)
		bm.notificationQueue = append(bm.notificationQueue, BackgroundNotification{
			TaskID:     taskID,
			Status:     status,
			Command:    utils.Truncate(command, 80),
			Preview:    preview,
			OutputFile: relOutputFile,
		})
	}
	bm.mu.Unlock()
}

func (bm *BackgroundManager) Check(taskID string) string {
	if taskID != "" {
		task, ok := bm.tasks[taskID]
		if !ok {
			return fmt.Sprintf("Error: Unknown task %s", taskID)
		}
		visible := map[string]string{
			"id":             task.ID,
			"status":         task.Status,
			"command":        task.Command,
			"result_preview": task.ResultPreview,
			"output_file":    task.OutputFile,
		}
		data, _ := json.MarshalIndent(visible, "", "  ")
		return string(data)
	}

	var lines []string
	for _, task := range bm.tasks {
		preview := task.ResultPreview
		if preview == "" {
			preview = "(running)"
		}
		lines = append(lines, fmt.Sprintf("%s: [%s] %s -> %s", task.ID, task.Status, utils.Truncate(task.Command, 60), preview))
	}

	if len(lines) == 0 {
		return "No background tasks."
	}
	return strings.Join(lines, "\n")
}

func (bm *BackgroundManager) DrainNotifications() []BackgroundNotification {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	notifs := make([]BackgroundNotification, len(bm.notificationQueue))
	copy(notifs, bm.notificationQueue)
	bm.notificationQueue = []BackgroundNotification{}
	return notifs
}

func (bm *BackgroundManager) DetectStalled() []string {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	var stalled []string
	now := time.Now().Unix()

	for taskID, task := range bm.tasks {
		if task.Status != "running" {
			continue
		}
		elapsed := now - int64(task.StartedAt)
		if elapsed > StallThresholdS {
			stalled = append(stalled, taskID)
		}
	}
	return stalled
}

type BackgroundRunTool struct {
	manager *BackgroundManager
}

func NewBackgroundRunTool(manager *BackgroundManager) tools.Tool {
	return &BackgroundRunTool{manager: manager}
}

func (t *BackgroundRunTool) Name() string {
	return "background_run"
}

func (t *BackgroundRunTool) Description() string {
	return "Run a command in background. Returns immediately with task ID."
}

func (t *BackgroundRunTool) Execute(ctx gocontext.Context, args map[string]interface{}) string {
	command := utils.GetStringFromMap(args, "command")
	if command == "" {
		return "Error: command is required"
	}
	return t.manager.Run(command)
}

func (t *BackgroundRunTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Shell command to run in background",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

type CheckBackgroundTool struct {
	manager *BackgroundManager
}

func NewCheckBackgroundTool(manager *BackgroundManager) tools.Tool {
	return &CheckBackgroundTool{manager: manager}
}

func (t *CheckBackgroundTool) Name() string {
	return "check_background"
}

func (t *CheckBackgroundTool) Description() string {
	return "Check status of background task. Use task_id to check specific task."
}

func (t *CheckBackgroundTool) Execute(ctx gocontext.Context, args map[string]interface{}) string {
	taskID := utils.GetStringFromMap(args, "task_id")
	return t.manager.Check(taskID)
}

func (t *CheckBackgroundTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional task ID to check specific task",
					},
				},
			},
		},
	}
}
