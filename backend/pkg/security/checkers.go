package security

import (
	"fmt"
	"regexp"
	"strings"

	"agent-base/pkg/utils"
)

var safeBashPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^ls\b`),
	regexp.MustCompile(`^cat\b`),
	regexp.MustCompile(`^head\b`),
	regexp.MustCompile(`^tail\b`),
	regexp.MustCompile(`^grep\b`),
	regexp.MustCompile(`^find\b`),
	regexp.MustCompile(`^tree\b`),
	regexp.MustCompile(`^git status\b`),
	regexp.MustCompile(`^git log\b`),
	regexp.MustCompile(`^git diff\b`),
	regexp.MustCompile(`^git branch\b`),
	regexp.MustCompile(`^git show\b`),
	regexp.MustCompile(`^npm list\b`),
	regexp.MustCompile(`^go list\b`),
	regexp.MustCompile(`^go test\b`),
	regexp.MustCompile(`^pwd\b`),
	regexp.MustCompile(`^which\b`),
	regexp.MustCompile(`^echo\b`),
	regexp.MustCompile(`^type\b`),
	regexp.MustCompile(`^uname\b`),
}

func isSafeBashCommand(command string) bool {
	trimmed := strings.TrimSpace(command)
	for _, pattern := range safeBashPatterns {
		if pattern.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// PermissionChecker is the interface for the Chain of Responsibility
type PermissionChecker interface {
	SetNext(checker PermissionChecker) PermissionChecker
	Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{}
}

// BasePermissionChecker provides the default next-chaining logic
type BasePermissionChecker struct {
	next PermissionChecker
}

func (b *BasePermissionChecker) SetNext(next PermissionChecker) PermissionChecker {
	b.next = next
	return next
}

func (b *BasePermissionChecker) CheckNext(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	if b.next != nil {
		return b.next.Check(pm, toolName, toolInput)
	}
	return map[string]interface{}{
		"behavior": "ask",
		"reason":   "No checker handled the request",
	}
}

// 1. PathSecurityChecker
type PathSecurityChecker struct {
	BasePermissionChecker
}

func (c *PathSecurityChecker) Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	path := utils.GetStringFromMap(toolInput, "path")
	if path != "" && !pm.isPathAllowed(path) {
		return map[string]interface{}{
			"behavior":        "ask",
			"reason":          fmt.Sprintf("Path outside workspace: %s. Grant access?", path),
			"needs_path_auth": true,
			"requested_path":  path,
		}
	}
	return c.CheckNext(pm, toolName, toolInput)
}

// 2. DenyRulesChecker
type DenyRulesChecker struct {
	BasePermissionChecker
}

func (c *DenyRulesChecker) Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	for _, rule := range pm.rules {
		if rule.Behavior != "deny" {
			continue
		}
		if pm.matchesRule(rule, toolName, toolInput) {
			return map[string]interface{}{
				"behavior": "deny",
				"reason":   fmt.Sprintf("Blocked by deny rule: %+v", rule),
			}
		}
	}
	return c.CheckNext(pm, toolName, toolInput)
}

// 3. BashSecurityChecker
type BashSecurityChecker struct {
	BasePermissionChecker
}

func (c *BashSecurityChecker) Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	if toolName == "bash" {
		command := utils.GetStringFromMap(toolInput, "command")

		failures := pm.bashValidator.Validate(command)
		if len(failures) > 0 {
			severe := map[string]bool{"sudo": true, "rm_rf": true}
			for _, f := range failures {
				if severe[f.Name] {
					desc := pm.bashValidator.DescribeFailures(command)
					return map[string]interface{}{
						"behavior": "deny",
						"reason":   fmt.Sprintf("Bash validator: %s", desc),
					}
				}
			}
			desc := pm.bashValidator.DescribeFailures(command)
			return map[string]interface{}{
				"behavior": "ask",
				"reason":   fmt.Sprintf("Bash validator flagged: %s", desc),
			}
		}

		if isSafeBashCommand(command) {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Safe read-only command: %s", command),
			}
		}
	}
	return c.CheckNext(pm, toolName, toolInput)
}

// 4. AllowRulesChecker
type AllowRulesChecker struct {
	BasePermissionChecker
}

func (c *AllowRulesChecker) Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	for _, rule := range pm.rules {
		if rule.Behavior != "allow" {
			continue
		}
		if pm.matchesRule(rule, toolName, toolInput) {
			pm.consecutiveDenials = 0
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Matched allow rule: %+v", rule),
			}
		}
	}
	return c.CheckNext(pm, toolName, toolInput)
}

// 5. GlobalModeChecker
type GlobalModeChecker struct {
	BasePermissionChecker
}

func (c *GlobalModeChecker) Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	if pm.mode == "plan" {
		if WriteTools[toolName] {
			return map[string]interface{}{
				"behavior": "deny",
				"reason":   fmt.Sprintf("Plan mode: write tool '%s' is blocked", toolName),
			}
		}
		if SystemTools[toolName] {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Plan mode: system tool '%s' allowed", toolName),
			}
		}
		if ReadOnlyTools[toolName] {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Plan mode: read-only tool '%s' allowed", toolName),
			}
		}
		return map[string]interface{}{
			"behavior": "ask",
			"reason":   fmt.Sprintf("Plan mode: unclassified tool '%s' needs confirmation", toolName),
		}
	}

	if pm.mode == "build" {
		if SystemTools[toolName] {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Build mode: system tool '%s' auto-approved", toolName),
			}
		}
		if ReadOnlyTools[toolName] {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Build mode: read-only tool '%s' auto-approved", toolName),
			}
		}
		if WriteTools[toolName] {
			return map[string]interface{}{
				"behavior": "allow",
				"reason":   fmt.Sprintf("Build mode: write tool '%s' auto-approved", toolName),
			}
		}
	}

	return c.CheckNext(pm, toolName, toolInput)
}

// 6. FallbackChecker
type FallbackChecker struct {
	BasePermissionChecker
}

func (c *FallbackChecker) Check(pm *PermissionManager, toolName string, toolInput map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"behavior": "ask",
		"reason":   fmt.Sprintf("No rule matched for %s, asking user", toolName),
	}
}
