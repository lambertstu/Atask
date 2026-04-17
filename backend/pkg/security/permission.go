package security

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"agent-base/pkg/utils"
)

const (
	PlanMode  = "plan"
	BuildMode = "build"
)

const TrustMarker = ".claude/.claude_trusted"

var Modes = []string{PlanMode, BuildMode}

var ReadOnlyTools = map[string]bool{
	"read_file":    true,
	"search_files": true,
	"grep_code":    true,
	"webfetch":     true,
}

var SystemTools = map[string]bool{
	"todo":              true,
	"task_create":       true,
	"task_update":       true,
	"delegate_subagent": true,
	"save_memory":       true,
	"load_skill":        true,
	"compact":           true,
	"task_list":         true,
	"task_get":          true,
	"check_background":  true,
	"cron_list":         true,
}

var WriteTools = map[string]bool{
	"write_file":     true,
	"edit_file":      true,
	"bash":           true,
	"background_run": true,
	"cron_create":    true,
	"cron_delete":    true,
}

type BlockingRequest struct {
	ToolName   string
	ToolInput  map[string]interface{}
	Decision   map[string]interface{}
	ResponseCh chan BlockingResponse
}

type BlockingResponse struct {
	Approved   bool
	AddAllowed string
}

type BashValidator struct {
	rules []struct {
		name    string
		pattern *regexp.Regexp
	}
}

func NewBashValidator() *BashValidator {
	v := &BashValidator{}
	v.rules = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"sudo", regexp.MustCompile(`\bsudo\b`)},
		{"rm_rf", regexp.MustCompile(`\brm\s+(-[a-zA-Z]*)?r`)},
		{"cmd_substitution", regexp.MustCompile(`\$\(`)},
		{"ifs_injection", regexp.MustCompile(`\bIFS\s*=`)},
	}
	return v
}

func (v *BashValidator) Validate(command string) []struct {
	Name    string
	Pattern string
} {
	var failures []struct {
		Name    string
		Pattern string
	}
	for _, rule := range v.rules {
		if rule.pattern.MatchString(command) {
			failures = append(failures, struct {
				Name    string
				Pattern string
			}{rule.name, rule.pattern.String()})
		}
	}
	return failures
}

func (v *BashValidator) IsSafe(command string) bool {
	return len(v.Validate(command)) == 0
}

func (v *BashValidator) DescribeFailures(command string) string {
	failures := v.Validate(command)
	if len(failures) == 0 {
		return "No security issues detected"
	}
	var parts []string
	for _, f := range failures {
		parts = append(parts, fmt.Sprintf("%s (pattern: %s)", f.Name, f.Pattern))
	}
	return "Security flags: " + strings.Join(parts, ", ")
}

type PermissionRule struct {
	Tool     string
	Path     string
	Content  string
	Behavior string
}

type PermissionManager struct {
	mode                  string
	rules                 []PermissionRule
	allowedDirs           []string
	workDir               string
	consecutiveDenials    int
	maxConsecutiveDenials int
	bashValidator         *BashValidator
	checkerChain          PermissionChecker
	blockedCallback       func(toolName string, toolInput map[string]interface{})
	blockingChan          chan BlockingRequest
}

func NewPermissionManager(mode string, workDir string) *PermissionManager {
	if !utils.Contains(Modes, mode) {
		mode = PlanMode
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		absWorkDir = workDir
	}

	pm := &PermissionManager{
		mode:    mode,
		workDir: absWorkDir,
		rules: []PermissionRule{
			{Tool: "bash", Content: "rm -rf /", Behavior: "deny"},
			{Tool: "bash", Content: "sudo *", Behavior: "deny"},
			{Tool: "read_file", Path: "*", Behavior: "allow"},
		},
		allowedDirs:           []string{},
		consecutiveDenials:    0,
		maxConsecutiveDenials: 3,
		bashValidator:         NewBashValidator(),
	}

	pathChecker := &PathSecurityChecker{}
	denyChecker := &DenyRulesChecker{}
	bashChecker := &BashSecurityChecker{}
	allowChecker := &AllowRulesChecker{}
	modeChecker := &GlobalModeChecker{}
	fallbackChecker := &FallbackChecker{}

	pathChecker.SetNext(denyChecker).
		SetNext(bashChecker).
		SetNext(allowChecker).
		SetNext(modeChecker).
		SetNext(fallbackChecker)

	pm.checkerChain = pathChecker

	return pm
}

func (p *PermissionManager) Check(toolName string, toolInput map[string]interface{}) map[string]interface{} {
	return p.checkerChain.Check(p, toolName, toolInput)
}

func (p *PermissionManager) SetMode(mode string) {
	if utils.Contains(Modes, mode) {
		p.mode = mode
	}
}

func (p *PermissionManager) AskUser(toolName string, toolInput map[string]interface{}) bool {
	fmt.Printf("\n\033[33m[SECURITY WARNING]\033[0m Agent wants to execute tool: \033[1m%s\033[0m\n", toolName)
	fmt.Printf("Arguments: %v\n", toolInput)
	fmt.Print("Do you want to allow this operation? (y/N): ")

	var response string
	fmt.Scanln(&response)

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		p.consecutiveDenials = 0
		return true
	}

	p.consecutiveDenials++
	if p.consecutiveDenials >= p.maxConsecutiveDenials {
		fmt.Printf("\n\033[31m[%d consecutive denials -- consider switching to plan mode]\033[0m\n", p.consecutiveDenials)
	}
	return false
}

func (p *PermissionManager) AskUserREPL(toolName string, toolInput map[string]interface{}, decision map[string]interface{}) bool {
	if decision["needs_path_auth"] == true {
		requestedPath := decision["requested_path"].(string)
		requestedDir := filepath.Dir(requestedPath)

		fmt.Printf("\n\033[33m[PATH AUTH]\033[0m Grant access to directory: \033[1m%s\033[0m? (y/N): ", requestedDir)
		var response string
		fmt.Scanln(&response)

		response = strings.TrimSpace(strings.ToLower(response))
		if response == "y" || response == "yes" {
			p.AddAllowedDir(requestedDir)
			p.consecutiveDenials = 0
			return true
		}

		p.consecutiveDenials++
		if p.consecutiveDenials >= p.maxConsecutiveDenials {
			fmt.Printf("\n\033[31m[%d consecutive denials -- consider switching to plan mode]\033[0m\n", p.consecutiveDenials)
		}
		return false
	}

	return p.AskUser(toolName, toolInput)
}

func (p *PermissionManager) AddRule(rule PermissionRule) {
	p.rules = append(p.rules, rule)
}

func (p *PermissionManager) matchesRule(rule PermissionRule, toolName string, toolInput map[string]interface{}) bool {
	if rule.Tool != "" && rule.Tool != "*" && rule.Tool != toolName {
		return false
	}
	if rule.Path != "" && rule.Path != "*" {
		path := utils.GetStringFromMap(toolInput, "path")
		if !utils.MatchGlob(path, rule.Path) {
			return false
		}
	}
	if rule.Content != "" {
		command := utils.GetStringFromMap(toolInput, "command")
		if !utils.MatchGlob(command, rule.Content) {
			return false
		}
	}
	return true
}

func (p *PermissionManager) isPathAllowed(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// 优先检查工作目录（自动授权）
	if p.workDir != "" && strings.HasPrefix(absPath, p.workDir) {
		return true
	}

	// 再检查用户授权的额外目录
	for _, allowed := range p.allowedDirs {
		if strings.HasPrefix(absPath, allowed) {
			return true
		}
	}

	return false
}

func (p *PermissionManager) AddAllowedDir(dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return
	}

	for _, existing := range p.allowedDirs {
		if existing == absDir {
			return
		}
	}

	p.allowedDirs = append(p.allowedDirs, absDir)
}

func (p *PermissionManager) GetAllowedDirs() []string {
	return p.allowedDirs
}

func (p *PermissionManager) SetBlockedCallback(cb func(toolName string, toolInput map[string]interface{})) {
	p.blockedCallback = cb
}

func (p *PermissionManager) SetBlockingChannel(ch chan BlockingRequest) {
	p.blockingChan = ch
}

func (p *PermissionManager) GetBlockingChannel() chan BlockingRequest {
	return p.blockingChan
}

func (p *PermissionManager) IsBlockingMode() bool {
	return p.blockingChan != nil
}

func (p *PermissionManager) WaitForDecision(ctx context.Context, responseCh chan BlockingResponse) (BlockingResponse, error) {
	select {
	case resp := <-responseCh:
		return resp, nil
	case <-ctx.Done():
		return BlockingResponse{Approved: false}, ctx.Err()
	}
}
