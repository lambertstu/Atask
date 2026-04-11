package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

type TaskRecord struct {
	ID          int    `json:"id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Status      string `json:"status"`
	BlockedBy   []int  `json:"blockedBy"`
	Blocks      []int  `json:"blocks"`
	Owner       string `json:"owner"`
}

type TaskManager struct {
	dir    string
	nextID int
}

func NewTaskManager(dir string) *TaskManager {
	tm := &TaskManager{
		dir:    dir,
		nextID: 1,
	}
	os.MkdirAll(dir, 0755)
	tm.nextID = tm.maxID() + 1
	return tm
}

func (tm *TaskManager) maxID() int {
	maxID := 0
	filepath.Walk(tm.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasPrefix(info.Name(), "task_") {
			return nil
		}

		idStr := strings.TrimSuffix(strings.TrimPrefix(info.Name(), "task_"), ".json")
		if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
			maxID = id
		}
		return nil
	})
	return maxID
}

func (tm *TaskManager) load(taskID int) (*TaskRecord, error) {
	path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", taskID))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("task %d not found", taskID)
	}

	var task TaskRecord
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (tm *TaskManager) save(task *TaskRecord) {
	path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", task.ID))
	data, _ := json.MarshalIndent(task, "", "  ")
	os.WriteFile(path, data, 0644)
}

func (tm *TaskManager) Create(subject, description string) string {
	task := &TaskRecord{
		ID:          tm.nextID,
		Subject:     subject,
		Description: description,
		Status:      "pending",
		BlockedBy:   []int{},
		Blocks:      []int{},
		Owner:       "",
	}
	tm.nextID++
	tm.save(task)

	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

func (tm *TaskManager) Get(taskID int) string {
	task, err := tm.load(taskID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

func (tm *TaskManager) Update(taskID int, status, owner string, addBlockedBy, addBlocks []int) string {
	task, err := tm.load(taskID)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if owner != "" {
		task.Owner = owner
	}

	if status != "" {
		validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "deleted": true}
		if !validStatuses[status] {
			return fmt.Sprintf("Error: invalid status: %s", status)
		}
		task.Status = status

		if status == "completed" {
			tm.clearDependency(taskID)
		}
	}

	if len(addBlockedBy) > 0 {
		task.BlockedBy = append(task.BlockedBy, addBlockedBy...)
		task.BlockedBy = utils.UniqueInts(task.BlockedBy)
	}

	if len(addBlocks) > 0 {
		task.Blocks = append(task.Blocks, addBlocks...)
		task.Blocks = utils.UniqueInts(task.Blocks)

		for _, blockedID := range addBlocks {
			blocked, err := tm.load(blockedID)
			if err != nil {
				continue
			}
			if !utils.ContainsInt(blocked.BlockedBy, taskID) {
				blocked.BlockedBy = append(blocked.BlockedBy, taskID)
				tm.save(blocked)
			}
		}
	}

	tm.save(task)
	data, _ := json.MarshalIndent(task, "", "  ")
	return string(data)
}

func (tm *TaskManager) clearDependency(completedID int) {
	filepath.Walk(tm.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var t TaskRecord
		if err := json.Unmarshal(data, &t); err != nil {
			return nil
		}

		if utils.ContainsInt(t.BlockedBy, completedID) {
			t.BlockedBy = utils.RemoveInt(t.BlockedBy, completedID)
			tm.save(&t)
		}
		return nil
	})
}

func (tm *TaskManager) ListAll() string {
	var tasks []*TaskRecord
	filepath.Walk(tm.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		data, _ := os.ReadFile(path)
		var task TaskRecord
		if err := json.Unmarshal(data, &task); err == nil {
			tasks = append(tasks, &task)
		}
		return nil
	})

	if len(tasks) == 0 {
		return "No tasks."
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	var lines []string
	for _, t := range tasks {
		var marker string
		switch t.Status {
		case "pending":
			marker = "[ ]"
		case "in_progress":
			marker = "[>]"
		case "completed":
			marker = "[x]"
		case "deleted":
			marker = "[-]"
		default:
			marker = "[?]"
		}

		line := fmt.Sprintf("%s #%d: %s", marker, t.ID, t.Subject)
		if t.Owner != "" {
			line += fmt.Sprintf(" owner=%s", t.Owner)
		}
		if len(t.BlockedBy) > 0 {
			line += fmt.Sprintf(" (blocked by: %v)", t.BlockedBy)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

type TaskCreateTool struct {
	manager *TaskManager
}

func NewTaskCreateTool(manager *TaskManager) tools.Tool {
	return &TaskCreateTool{manager: manager}
}

func (t *TaskCreateTool) Name() string {
	return "task_create"
}

func (t *TaskCreateTool) Description() string {
	return "Create a new task with subject and description."
}

func (t *TaskCreateTool) Execute(ctx context.Context, args map[string]interface{}) string {
	subject := utils.GetStringFromMap(args, "subject")
	description := utils.GetStringFromMap(args, "description")
	if subject == "" {
		return "Error: subject is required"
	}
	return t.manager.Create(subject, description)
}

func (t *TaskCreateTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"subject": map[string]interface{}{
						"type":        "string",
						"description": "Task subject/title",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Task description",
					},
				},
				"required": []string{"subject"},
			},
		},
	}
}

type TaskUpdateTool struct {
	manager *TaskManager
}

func NewTaskUpdateTool(manager *TaskManager) tools.Tool {
	return &TaskUpdateTool{manager: manager}
}

func (t *TaskUpdateTool) Name() string {
	return "task_update"
}

func (t *TaskUpdateTool) Description() string {
	return "Update task status, owner, and dependencies."
}

func (t *TaskUpdateTool) Execute(ctx context.Context, args map[string]interface{}) string {
	taskID := utils.GetIntFromMap(args, "task_id")
	status := utils.GetStringFromMap(args, "status")
	owner := utils.GetStringFromMap(args, "owner")
	addBlockedBy := utils.GetIntArrayFromMap(args, "add_blocked_by")
	addBlocks := utils.GetIntArrayFromMap(args, "add_blocks")

	if taskID == 0 {
		return "Error: task_id is required"
	}
	return t.manager.Update(taskID, status, owner, addBlockedBy, addBlocks)
}

func (t *TaskUpdateTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "integer",
						"description": "Task ID",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "New status: pending, in_progress, completed, deleted",
					},
					"owner": map[string]interface{}{
						"type":        "string",
						"description": "Task owner",
					},
					"add_blocked_by": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "integer"},
						"description": "IDs of tasks blocking this one",
					},
					"add_blocks": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "integer"},
						"description": "IDs of tasks this one blocks",
					},
				},
				"required": []string{"task_id"},
			},
		},
	}
}

type TaskListTool struct {
	manager *TaskManager
}

func NewTaskListTool(manager *TaskManager) tools.Tool {
	return &TaskListTool{manager: manager}
}

func (t *TaskListTool) Name() string {
	return "task_list"
}

func (t *TaskListTool) Description() string {
	return "List all tasks."
}

func (t *TaskListTool) Execute(ctx context.Context, args map[string]interface{}) string {
	return t.manager.ListAll()
}

func (t *TaskListTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

type TaskGetTool struct {
	manager *TaskManager
}

func NewTaskGetTool(manager *TaskManager) tools.Tool {
	return &TaskGetTool{manager: manager}
}

func (t *TaskGetTool) Name() string {
	return "task_get"
}

func (t *TaskGetTool) Description() string {
	return "Get details of a specific task."
}

func (t *TaskGetTool) Execute(ctx context.Context, args map[string]interface{}) string {
	taskID := utils.GetIntFromMap(args, "task_id")
	if taskID == 0 {
		return "Error: task_id is required"
	}
	return t.manager.Get(taskID)
}

func (t *TaskGetTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "integer",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
	}
}
