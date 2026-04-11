package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"agent-base/pkg/utils"
)

const TrustMarker = ".claude/.claude_trusted"

var Modes = []string{"default", "plan", "auto"}

var ReadOnlyTools = map[string]bool{
	"read_file":     true,
	"bash_readonly": true,
}

var WriteTools = map[string]bool{
	"write_file": true,
	"edit_file":  true,
	"bash":       true,
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
		{"shell_metachar", regexp.MustCompile(`[;&|\` + "`" + `$]`)},
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
}

func NewPermissionManager(mode string, workDir string) *PermissionManager {
	if !utils.Contains(Modes, mode) {
		mode = "default"
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		absWorkDir = workDir
	}

	return &PermissionManager{
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
}

func (p *PermissionManager) Check(toolName string, toolInput map[string]interface{}) map[string]interface{} {
	path := utils.GetStringFromMap(toolInput, "path")
	if path != "" && !p.isPathAllowed(path) {
		return map[string]interface{}{
			"behavior":        "ask",
			"reason":          fmt.Sprintf("Path outside workspace: %s. Grant access?", path),
			"needs_path_auth": true,
			"requested_path":  path,
		}
	}

	for _, rule := range p.rules {
		if rule.Behavior != "allow" {
			continue
		}
		if p.matchesRule(rule, toolName, toolInput) {
			p.consecutiveDenials = 0
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Matched allow rule: %+v", rule),
			}
		}
	}

	if toolName == "bash" {
		command := utils.GetStringFromMap(toolInput, "command")
		failures := p.bashValidator.Validate(command)
		if len(failures) > 0 {
			severe := map[string]bool{"sudo": true, "rm_rf": true}
			for _, f := range failures {
				if severe[f.Name] {
					desc := p.bashValidator.DescribeFailures(command)
					return map[string]interface{}{
						"behavior": "deny",
						"reason":   fmt.Sprintf("Bash validator: %s", desc),
					}
				}
			}
			desc := p.bashValidator.DescribeFailures(command)
			return map[string]interface{}{
				"behavior": "ask",
				"reason":   fmt.Sprintf("Bash validator flagged: %s", desc),
			}
		}
	}

	for _, rule := range p.rules {
		if rule.Behavior != "deny" {
			continue
		}
		if p.matchesRule(rule, toolName, toolInput) {
			return map[string]interface{}{
				"behavior": "deny",
				"reason":   fmt.Sprintf("Blocked by deny rule: %+v", rule),
			}
		}
	}

	if p.mode == "plan" {
		if WriteTools[toolName] {
			return map[string]interface{}{
				"behavior": "deny",
				"reason":   "Plan mode: write operations are blocked",
			}
		}
		return map[string]interface{}{
			"behavior": "allow",
			"reason":   "Plan mode: read-only allowed",
		}
	}

	if p.mode == "auto" {
		if ReadOnlyTools[toolName] {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   "Auto mode: read-only tool auto-approved",
			}
		}
	}

	return map[string]interface{}{
		"behavior": "ask",
		"reason":   fmt.Sprintf("No rule matched for %s, asking user", toolName),
	}
}

func (p *PermissionManager) SetMode(mode string) {
	if utils.Contains(Modes, mode) {
		p.mode = mode
	}
}

func (p *PermissionManager) AskUser(toolName string, toolInput map[string]interface{}) bool {
	p.consecutiveDenials++
	if p.consecutiveDenials >= p.maxConsecutiveDenials {
		fmt.Printf("[%d consecutive denials -- consider switching to plan mode]\n", p.consecutiveDenials)
	}
	return false
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
