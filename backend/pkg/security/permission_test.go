package security

import (
	"path/filepath"
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
)

func TestNewPermissionManager(t *testing.T) {
	tests := []struct {
		mode         string
		expectedMode string
	}{
		{"plan", "plan"},
		{"build", "build"},
		{"invalid", "build"},
		{"", "build"},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			pm := NewPermissionManager(tt.mode, "/workspace")
			assert.Equal(t, tt.expectedMode, pm.mode)
		})
	}
}

func TestNewPermissionManager_WorkDir(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	pm := NewPermissionManager("build", tempDir.Path)
	assert.Equal(t, tempDir.Path, pm.workDir)
}

func TestCheck_PermissionMatrix(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	insidePath := filepath.Join(tempDir.Path, "test.go")
	outsidePath := "/etc/passwd"

	tests := []struct {
		name                  string
		tool                  string
		mode                  string
		toolInput             map[string]interface{}
		expectedBehavior      string
		expectedNeedsPathAuth bool
	}{
		{"read_file inside workspace", "read_file", "build", map[string]interface{}{"path": insidePath}, "allow", false},
		{"read_file outside workspace", "read_file", "build", map[string]interface{}{"path": outsidePath}, "ask", true},
		{"write_file inside workspace", "write_file", "build", map[string]interface{}{"path": insidePath}, "ask", false},
		{"write_file in plan mode", "write_file", "plan", map[string]interface{}{"path": insidePath}, "deny", false},
		{"read_file in plan mode", "read_file", "plan", map[string]interface{}{"path": insidePath}, "allow", false},
		{"bash safe command in build mode", "bash", "build", map[string]interface{}{"command": "ls"}, "allow", false},
		{"bash dangerous command in build mode", "bash", "build", map[string]interface{}{"command": "sudo ls"}, "deny", false},
		{"task_list in plan mode", "task_list", "plan", map[string]interface{}{}, "allow", false},
		{"task_create in plan mode", "task_create", "plan", map[string]interface{}{}, "deny", false},
		{"unknown tool", "unknown_tool", "build", map[string]interface{}{}, "ask", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := NewPermissionManager(tt.mode, tempDir.Path)
			result := pm.Check(tt.tool, tt.toolInput)
			assert.Equal(t, tt.expectedBehavior, result["behavior"])
			if tt.expectedNeedsPathAuth {
				assert.Equal(t, true, result["needs_path_auth"])
			}
		})
	}
}

func TestCheck_BashDangerousCommands(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")

	tests := []struct {
		command  string
		expected string
	}{
		{"sudo rm -rf /", "deny"},
		{"rm -rf /", "deny"},
		{"rm -rf --no-preserve-root /", "deny"},
		{"sudo ls", "deny"},
		{"ls; rm -rf /", "deny"},
		{"rm -rf /home/user", "deny"},
		{"ls $(whoami)", "ask"},
		{"ls; echo done", "ask"},
		{"ls | grep test", "ask"},
		{"ls && echo done", "ask"},
		{"ls", "allow"},
		{"git status", "allow"},
		{"cat file.txt", "allow"},
		{"pwd", "allow"},
		{"go test", "allow"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := pm.Check("bash", map[string]interface{}{"command": tt.command})
			assert.Equal(t, tt.expected, result["behavior"])
		})
	}
}

func TestCheck_PathAuthorization(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	pm := NewPermissionManager("build", tempDir.Path)

	outsidePath := "/etc/passwd"

	t.Logf("workDir: %s", pm.workDir)
	t.Logf("isPathAllowed('/etc/passwd'): %v", pm.isPathAllowed("/etc/passwd"))

	result := pm.Check("read_file", map[string]interface{}{"path": outsidePath})
	t.Logf("First check result: %v", result)
	assert.Equal(t, "ask", result["behavior"])
	assert.Equal(t, true, result["needs_path_auth"])

	pm.AddAllowedDir("/etc")

	result = pm.Check("read_file", map[string]interface{}{"path": outsidePath})
	t.Logf("Second check result: %v", result)
	assert.Equal(t, "allow", result["behavior"])
}

func TestSetMode(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")

	pm.SetMode("plan")
	assert.Equal(t, "plan", pm.mode)

	pm.SetMode("build")
	assert.Equal(t, "build", pm.mode)

	pm.SetMode("invalid")
	assert.Equal(t, "build", pm.mode)
}

func TestIsPathAllowed(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	pm := NewPermissionManager("build", tempDir.Path)

	tests := []struct {
		path     string
		expected bool
	}{
		{filepath.Join(tempDir.Path, "test.go"), true},
		{filepath.Join(tempDir.Path, "internal/engine/test.go"), true},
		{"/etc/passwd", false},
		{"../test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := pm.isPathAllowed(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPathAllowed_WithAllowedDirs(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	pm := NewPermissionManager("build", tempDir.Path)

	pm.AddAllowedDir("/etc")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/etc/passwd", true},
		{"/etc/hosts", true},
		{"/var/log/test.log", false},
		{filepath.Join(tempDir.Path, "test.go"), true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := pm.isPathAllowed(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddAllowedDir(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")

	pm.AddAllowedDir("/tmp")
	assert.Equal(t, []string{"/tmp"}, pm.GetAllowedDirs())

	pm.AddAllowedDir("/var")
	assert.Equal(t, []string{"/tmp", "/var"}, pm.GetAllowedDirs())

	pm.AddAllowedDir("/tmp")
	assert.Equal(t, []string{"/tmp", "/var"}, pm.GetAllowedDirs())
}

func TestGetAllowedDirs(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")
	assert.Equal(t, []string{}, pm.GetAllowedDirs())

	pm.AddAllowedDir("/tmp")
	assert.Equal(t, []string{"/tmp"}, pm.GetAllowedDirs())
}

func TestAskUser(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")

	result := pm.AskUser("bash", map[string]interface{}{"command": "ls"})
	assert.False(t, result)
	assert.Equal(t, 1, pm.consecutiveDenials)

	pm.AskUser("bash", map[string]interface{}{})
	pm.AskUser("bash", map[string]interface{}{})
	assert.Equal(t, 3, pm.consecutiveDenials)
}

func TestMatchesRule(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")

	tests := []struct {
		rule     PermissionRule
		tool     string
		input    map[string]interface{}
		expected bool
	}{
		{PermissionRule{Tool: "bash"}, "bash", map[string]interface{}{}, true},
		{PermissionRule{Tool: "bash"}, "read_file", map[string]interface{}{}, false},
		{PermissionRule{Tool: "*"}, "bash", map[string]interface{}{}, true},
		{PermissionRule{Path: "*"}, "read_file", map[string]interface{}{"path": "test.go"}, true},
		{PermissionRule{Path: "test*"}, "read_file", map[string]interface{}{"path": "test.go"}, true},
		{PermissionRule{Content: "rm -rf /"}, "bash", map[string]interface{}{"command": "rm -rf /"}, true},
		{PermissionRule{Content: "sudo *"}, "bash", map[string]interface{}{"command": "sudo ls"}, true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := pm.matchesRule(tt.rule, tt.tool, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddRule(t *testing.T) {
	pm := NewPermissionManager("build", "/workspace")

	rule := PermissionRule{
		Tool:     "write_file",
		Path:     "safe/*",
		Behavior: "allow",
	}

	pm.AddRule(rule)

	assert.Len(t, pm.rules, 4)
}
