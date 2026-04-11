package planning

import (
	"context"
	"fmt"
	"strings"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

const (
	ProcessingStatus = "processing"
	PendingStatus    = "pending"
	CompleteStatus   = "complete"
)

type TodoItem struct {
	ID     string
	Text   string
	Status string
}

type TodoManager struct {
	items []TodoItem
}

func NewTodoManager() *TodoManager {
	return &TodoManager{
		items: []TodoItem{},
	}
}

func (m *TodoManager) Update(rawItems interface{}) (string, error) {
	var validated []TodoItem
	var processingCounter int

	items, ok := rawItems.([]interface{})
	if !ok {
		return "", fmt.Errorf("items must be an array")
	}

	for _, rawItem := range items {
		item, ok := rawItem.(map[string]interface{})
		if !ok {
			continue
		}

		status := utils.GetStringFromMap(item, "status")
		if status == "" {
			status = PendingStatus
		}

		if status == ProcessingStatus {
			processingCounter++
		}

		id := utils.GetStringFromMap(item, "id")
		text := utils.GetStringFromMap(item, "text")

		validated = append(validated, TodoItem{
			ID:     id,
			Text:   text,
			Status: status,
		})
	}

	if processingCounter > 1 {
		return "", fmt.Errorf("only one task processing")
	}

	m.items = validated
	return m.Render(), nil
}

func (m *TodoManager) Render() string {
	if len(m.items) == 0 {
		return "no todo list"
	}

	var completeCounter int
	var lines []string

	for _, item := range m.items {
		var marker string
		switch item.Status {
		case PendingStatus:
			marker = "[ ]"
		case ProcessingStatus:
			marker = "[>]"
		case CompleteStatus:
			marker = "[√]"
			completeCounter++
		}
		lines = append(lines, fmt.Sprintf("%s #%s:%s", marker, item.ID, item.Text))
	}

	lines = append(lines, fmt.Sprintf("\n(%d/%d complete)", completeCounter, len(m.items)))
	return strings.Join(lines, "\n")
}

type TodoTool struct {
	manager *TodoManager
}

func NewTodoTool() *TodoTool {
	return &TodoTool{manager: NewTodoManager()}
}

func NewTodoToolWithManager(manager *TodoManager) *TodoTool {
	return &TodoTool{manager: manager}
}

func (t *TodoTool) Name() string {
	return "todo"
}

func (t *TodoTool) Description() string {
	return "Update task list. Track progress on multi-step tasks."
}

func (t *TodoTool) Execute(ctx context.Context, args map[string]interface{}) string {
	items := args["items"]
	update, err := t.manager.Update(items)
	if err != nil {
		return err.Error()
	}
	return update
}

func (t *TodoTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"id": map[string]interface{}{
									"type": "string",
								},
								"text": map[string]interface{}{
									"type": "string",
								},
								"status": map[string]interface{}{
									"type": "string",
									"enum": []string{PendingStatus, ProcessingStatus, CompleteStatus},
								},
							},
							"required": []string{"id", "text", "status"},
						},
					},
				},
				"required": []string{"items"},
			},
		},
	}
}

func RegisterTodoTool(registry tools.ToolRegistry, manager *TodoManager) {
	registry.Register(NewTodoToolWithManager(manager))
}
