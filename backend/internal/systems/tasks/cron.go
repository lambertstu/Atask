package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"agent-base/internal/tools"
	"agent-base/pkg/utils"

	"github.com/sashabaranov/go-openai"
)

const AutoExpiryDays = 7

type ScheduledTask struct {
	ID        string  `json:"id"`
	Cron      string  `json:"cron"`
	Prompt    string  `json:"prompt"`
	Recurring bool    `json:"recurring"`
	Durable   bool    `json:"durable"`
	CreatedAt float64 `json:"created_at"`
	LastFired float64 `json:"last_fired,omitempty"`
}

type CronScheduler struct {
	workDir         string
	tasks           []ScheduledTask
	queue           []string
	mu              sync.Mutex
	stopEvent       bool
	lastCheckMinute int
}

func NewCronScheduler() *CronScheduler {
	workDir, _ := os.Getwd()
	return &CronScheduler{
		workDir: workDir,
		tasks:   []ScheduledTask{},
		queue:   []string{},
	}
}

func NewCronSchedulerWithWorkDir(workDir string) *CronScheduler {
	return &CronScheduler{
		workDir: workDir,
		tasks:   []ScheduledTask{},
		queue:   []string{},
	}
}

func (cs *CronScheduler) Start() {
	cs.loadDurable()
	go cs.checkLoop()
	if len(cs.tasks) > 0 {
		fmt.Printf("[Cron] Loaded %d scheduled tasks\n", len(cs.tasks))
	}
}

func (cs *CronScheduler) Stop() {
	cs.mu.Lock()
	cs.stopEvent = true
	cs.mu.Unlock()
}

func (cs *CronScheduler) Create(cronExpr, prompt string, recurring, durable bool) string {
	taskID := fmt.Sprintf("%d", time.Now().UnixNano()/1000000)[:8]

	task := ScheduledTask{
		ID:        taskID,
		Cron:      cronExpr,
		Prompt:    prompt,
		Recurring: recurring,
		Durable:   durable,
		CreatedAt: float64(time.Now().Unix()),
	}

	cs.mu.Lock()
	cs.tasks = append(cs.tasks, task)
	cs.mu.Unlock()

	if durable {
		cs.saveDurable()
	}

	mode := "recurring"
	if !recurring {
		mode = "one-shot"
	}
	store := "durable"
	if !durable {
		store = "session-only"
	}

	return fmt.Sprintf("Created task %s (%s, %s): cron=%s", taskID, mode, store, cronExpr)
}

func (cs *CronScheduler) Delete(taskID string) string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var newTasks []ScheduledTask
	found := false
	for _, task := range cs.tasks {
		if task.ID == taskID {
			found = true
			continue
		}
		newTasks = append(newTasks, task)
	}

	if !found {
		return fmt.Sprintf("Task %s not found", taskID)
	}

	cs.tasks = newTasks
	cs.saveDurable()
	return fmt.Sprintf("Deleted task %s", taskID)
}

func (cs *CronScheduler) ListTasks() string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if len(cs.tasks) == 0 {
		return "No scheduled tasks."
	}

	var lines []string
	for _, task := range cs.tasks {
		mode := "recurring"
		if !task.Recurring {
			mode = "one-shot"
		}
		store := "durable"
		if !task.Durable {
			store = "session"
		}
		ageHours := (float64(time.Now().Unix()) - task.CreatedAt) / 3600
		lines = append(lines, fmt.Sprintf("  %s  %s  [%s/%s] (%.1fh old): %s", task.ID, task.Cron, mode, store, ageHours, utils.Truncate(task.Prompt, 60)))
	}

	return strings.Join(lines, "\n")
}

func (cs *CronScheduler) DrainNotifications() []string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	notifs := make([]string, len(cs.queue))
	copy(notifs, cs.queue)
	cs.queue = []string{}
	return notifs
}

func (cs *CronScheduler) checkLoop() {
	for {
		cs.mu.Lock()
		if cs.stopEvent {
			cs.mu.Unlock()
			return
		}
		cs.mu.Unlock()

		now := time.Now()
		currentMinute := now.Hour()*60 + now.Minute()

		if currentMinute != cs.lastCheckMinute {
			cs.lastCheckMinute = currentMinute
			cs.checkTasks(now)
		}

		time.Sleep(1 * time.Second)
	}
}

func (cs *CronScheduler) checkTasks(now time.Time) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	var expired []string
	var firedOneshots []string

	for _, task := range cs.tasks {
		ageDays := (float64(time.Now().Unix()) - task.CreatedAt) / 86400
		if task.Recurring && ageDays > AutoExpiryDays {
			expired = append(expired, task.ID)
			continue
		}

		if cronMatches(task.Cron, now) {
			notification := fmt.Sprintf("[Scheduled task %s]: %s", task.ID, task.Prompt)
			cs.queue = append(cs.queue, notification)
			task.LastFired = float64(now.Unix())
			fmt.Printf("[Cron] Fired: %s\n", task.ID)

			if !task.Recurring {
				firedOneshots = append(firedOneshots, task.ID)
			}
		}
	}

	if len(expired) > 0 || len(firedOneshots) > 0 {
		removeIDs := make(map[string]bool)
		for _, id := range expired {
			removeIDs[id] = true
			fmt.Printf("[Cron] Auto-expired: %s (older than %d days)\n", id, AutoExpiryDays)
		}
		for _, id := range firedOneshots {
			removeIDs[id] = true
			fmt.Printf("[Cron] One-shot completed and removed: %s\n", id)
		}

		var newTasks []ScheduledTask
		for _, task := range cs.tasks {
			if !removeIDs[task.ID] {
				newTasks = append(newTasks, task)
			}
		}
		cs.tasks = newTasks
		cs.saveDurable()
	}
}

func (cs *CronScheduler) loadDurable() {
	tasksFile := filepath.Join(cs.workDir, ".claude", "scheduled_tasks.json")
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		return
	}

	var tasks []ScheduledTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		fmt.Printf("[Cron] Error loading tasks: %v\n", err)
		return
	}

	for _, task := range tasks {
		if task.Durable {
			cs.tasks = append(cs.tasks, task)
		}
	}
}

func (cs *CronScheduler) saveDurable() {
	var durable []ScheduledTask
	for _, task := range cs.tasks {
		if task.Durable {
			durable = append(durable, task)
		}
	}

	tasksFile := filepath.Join(cs.workDir, ".claude", "scheduled_tasks.json")
	os.MkdirAll(filepath.Dir(tasksFile), 0755)
	data, _ := json.MarshalIndent(durable, "", "  ")
	os.WriteFile(tasksFile, data, 0644)
}

func cronMatches(expr string, dt time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}

	minute := dt.Minute()
	hour := dt.Hour()
	day := dt.Day()
	month := int(dt.Month())
	dow := int(dt.Weekday())

	values := []int{minute, hour, day, month, dow}
	ranges := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}

	for i, field := range fields {
		if !fieldMatches(field, values[i], ranges[i][0], ranges[i][1]) {
			return false
		}
	}

	return true
}

func fieldMatches(field string, value, lo, hi int) bool {
	if field == "*" {
		return true
	}

	parts := strings.Split(field, ",")
	for _, part := range parts {
		step := 1
		if strings.Contains(part, "/") {
			split := strings.Split(part, "/")
			part = split[0]
			if s, err := strconv.Atoi(split[1]); err == nil {
				step = s
			}
		}

		if part == "*" {
			if (value-lo)%step == 0 {
				return true
			}
		} else if strings.Contains(part, "-") {
			split := strings.Split(part, "-")
			start, _ := strconv.Atoi(split[0])
			end, _ := strconv.Atoi(split[1])
			if start <= value && value <= end && (value-start)%step == 0 {
				return true
			}
		} else {
			if v, err := strconv.Atoi(part); err == nil && v == value {
				return true
			}
		}
	}

	return false
}

type CronCreateTool struct {
	scheduler *CronScheduler
}

func NewCronCreateTool(scheduler *CronScheduler) tools.Tool {
	return &CronCreateTool{scheduler: scheduler}
}

func (t *CronCreateTool) Name() string {
	return "cron_create"
}

func (t *CronCreateTool) Description() string {
	return "Create a scheduled task. Use cron expression (e.g., '0 9 * * 1' for every Monday 9am)."
}

func (t *CronCreateTool) Execute(ctx context.Context, args map[string]interface{}) string {
	cronExpr := utils.GetStringFromMap(args, "cron")
	prompt := utils.GetStringFromMap(args, "prompt")
	recurring := utils.GetBoolFromMap(args, "recurring")
	durable := utils.GetBoolFromMap(args, "durable")

	if cronExpr == "" || prompt == "" {
		return "Error: cron and prompt are required"
	}
	return t.scheduler.Create(cronExpr, prompt, recurring, durable)
}

func (t *CronCreateTool) Schema() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cron": map[string]interface{}{
						"type":        "string",
						"description": "Cron expression (minute hour day month weekday)",
					},
					"prompt": map[string]interface{}{
						"type":        "string",
						"description": "Prompt to execute when triggered",
					},
					"recurring": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether task recurs (default false)",
					},
					"durable": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether task persists across sessions (default false)",
					},
				},
				"required": []string{"cron", "prompt"},
			},
		},
	}
}

type CronDeleteTool struct {
	scheduler *CronScheduler
}

func NewCronDeleteTool(scheduler *CronScheduler) tools.Tool {
	return &CronDeleteTool{scheduler: scheduler}
}

func (t *CronDeleteTool) Name() string {
	return "cron_delete"
}

func (t *CronDeleteTool) Description() string {
	return "Delete a scheduled task by ID."
}

func (t *CronDeleteTool) Execute(ctx context.Context, args map[string]interface{}) string {
	taskID := utils.GetStringFromMap(args, "task_id")
	if taskID == "" {
		return "Error: task_id is required"
	}
	return t.scheduler.Delete(taskID)
}

func (t *CronDeleteTool) Schema() openai.Tool {
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
						"description": "ID of the scheduled task",
					},
				},
				"required": []string{"task_id"},
			},
		},
	}
}

type CronListTool struct {
	scheduler *CronScheduler
}

func NewCronListTool(scheduler *CronScheduler) tools.Tool {
	return &CronListTool{scheduler: scheduler}
}

func (t *CronListTool) Name() string {
	return "cron_list"
}

func (t *CronListTool) Description() string {
	return "List all scheduled tasks."
}

func (t *CronListTool) Execute(ctx context.Context, args map[string]interface{}) string {
	return t.scheduler.ListTasks()
}

func (t *CronListTool) Schema() openai.Tool {
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
