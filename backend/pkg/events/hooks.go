package events

import (
	gocontext "context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"agent-base/pkg/utils"
)

const HookTimeout = 30 * time.Second

var HookEvents = []string{"PreToolUse", "PostToolUse", "SessionStart"}

type HookDefinition struct {
	Matcher string `json:"matcher"`
	Command string `json:"command"`
}

type HookConfig struct {
	Hooks map[string][]HookDefinition `json:"hooks"`
}

type HookResult struct {
	Blocked            bool
	BlockReason        string
	Messages           []string
	PermissionOverride string
}

type HookManager struct {
	hooks   map[string][]HookDefinition
	workDir string
	sdkMode bool
}

func NewHookManager(workDir string, sdkMode bool) *HookManager {
	hm := &HookManager{
		hooks:   make(map[string][]HookDefinition),
		workDir: workDir,
		sdkMode: sdkMode,
	}

	configPath := filepath.Join(workDir, ".hooks.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var config HookConfig
		if err := json.Unmarshal(data, &config); err == nil {
			for _, event := range HookEvents {
				if hooks, ok := config.Hooks[event]; ok {
					hm.hooks[event] = hooks
				}
			}
			fmt.Printf("[Hooks loaded from %s]\n", configPath)
		}
	}

	return hm
}

func (hm *HookManager) RunHooks(event string, context map[string]interface{}) HookResult {
	result := HookResult{
		Blocked:  false,
		Messages: []string{},
	}

	hooks, ok := hm.hooks[event]
	if !ok {
		return result
	}

	for _, hook := range hooks {
		if hook.Matcher != "" && hook.Matcher != "*" {
			toolName := utils.GetStringFromMap(context, "tool_name")
			if hook.Matcher != toolName {
				continue
			}
		}

		if hook.Command == "" {
			continue
		}

		env := os.Environ()
		env = append(env, fmt.Sprintf("HOOK_EVENT=%s", event))

		if toolName := utils.GetStringFromMap(context, "tool_name"); toolName != "" {
			env = append(env, fmt.Sprintf("HOOK_TOOL_NAME=%s", toolName))
		}

		if toolInput := utils.GetMapFromMap(context, "tool_input"); toolInput != nil {
			inputJSON, _ := json.Marshal(toolInput)
			inputStr := string(inputJSON)
			if len(inputStr) > 10000 {
				inputStr = inputStr[:10000]
			}
			env = append(env, fmt.Sprintf("HOOK_TOOL_INPUT=%s", inputStr))
		}

		if toolOutput := utils.GetStringFromMap(context, "tool_output"); toolOutput != "" {
			output := toolOutput
			if len(output) > 10000 {
				output = output[:10000]
			}
			env = append(env, fmt.Sprintf("HOOK_TOOL_OUTPUT=%s", output))
		}

		ctx, cancel := gocontext.WithTimeout(gocontext.Background(), HookTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "bash", "-c", hook.Command)
		cmd.Dir = hm.workDir
		cmd.Env = env

		output, err := cmd.CombinedOutput()
		if ctx.Err() == gocontext.DeadlineExceeded {
			fmt.Printf("[hook:%s] Timeout (%v)\n", event, HookTimeout)
			continue
		}
		if err != nil && cmd.ProcessState == nil {
			fmt.Printf("[hook:%s] Error: %v\n", event, err)
			continue
		}

		combinedOutput := strings.TrimSpace(string(output))

		if cmd.ProcessState != nil {
			exitCode := cmd.ProcessState.ExitCode()
			switch exitCode {
			case 0:
				if combinedOutput != "" {
					fmt.Printf("[hook:%s] %s\n", event, utils.Truncate(combinedOutput, 100))
				}
				var hookOutput map[string]interface{}
				if err := json.Unmarshal([]byte(combinedOutput), &hookOutput); err == nil {
					if updatedInput := utils.GetMapFromMap(hookOutput, "updatedInput"); updatedInput != nil {
						context["tool_input"] = updatedInput
					}
					if additionalCtx := utils.GetStringFromMap(hookOutput, "additionalContext"); additionalCtx != "" {
						result.Messages = append(result.Messages, additionalCtx)
					}
					if permDecision := utils.GetStringFromMap(hookOutput, "permissionDecision"); permDecision != "" {
						result.PermissionOverride = permDecision
					}
				}
			case 1:
				result.Blocked = true
				reason := combinedOutput
				if reason == "" {
					reason = "Blocked by hook"
				}
				result.BlockReason = reason
				fmt.Printf("[hook:%s] BLOCKED: %s\n", event, utils.Truncate(reason, 200))
			case 2:
				msg := combinedOutput
				if msg != "" {
					result.Messages = append(result.Messages, msg)
					fmt.Printf("[hook:%s] INJECT: %s\n", event, utils.Truncate(msg, 200))
				}
			}
		}
	}

	return result
}
