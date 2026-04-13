package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBashValidator(t *testing.T) {
	v := NewBashValidator()
	assert.NotNil(t, v)
	assert.Len(t, v.rules, 4)

	expectedRules := []string{"sudo", "rm_rf", "cmd_substitution", "ifs_injection"}
	for i, expected := range expectedRules {
		assert.Equal(t, expected, v.rules[i].name)
	}
}

func TestBashValidator_Validate(t *testing.T) {
	v := NewBashValidator()

	tests := []struct {
		command  string
		expected []string
	}{
		{"ls", nil},
		{"ls; rm -rf /", []string{"rm_rf"}},
		{"sudo ls", []string{"sudo"}},
		{"rm -rf /", []string{"rm_rf"}},
		{"rm -rf --no-preserve-root /", []string{"rm_rf"}},
		{"rm -rf /home/user", []string{"rm_rf"}},
		{"ls $(whoami)", []string{"cmd_substitution"}},
		{"ls | grep test", nil},
		{"ls && echo done", nil},
		{"ls || echo fail", nil},
		{"echo `pwd`", nil},
		{"IFS= read -r line", []string{"ifs_injection"}},
		{"cat file", nil},
		{"echo 'hello world'", nil},
		{"git status", nil},
		{"npm run test", nil},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			failures := v.Validate(tt.command)
			var failureNames []string
			for _, f := range failures {
				failureNames = append(failureNames, f.Name)
			}
			assert.Equal(t, tt.expected, failureNames)
		})
	}
}

func TestBashValidator_IsSafe(t *testing.T) {
	v := NewBashValidator()

	tests := []struct {
		command  string
		expected bool
	}{
		{"ls", true},
		{"cat file.txt", true},
		{"git status", true},
		{"npm install", true},
		{"sudo ls", false},
		{"rm -rf /", false},
		{"ls; cat file", true},
		{"ls $(pwd)", false},
		{"IFS= read", false},
		{"ls | grep test", true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			assert.Equal(t, tt.expected, v.IsSafe(tt.command))
		})
	}
}

func TestBashValidator_DescribeFailures(t *testing.T) {
	v := NewBashValidator()

	tests := []struct {
		command  string
		contains string
	}{
		{"ls", "No security issues detected"},
		{"sudo rm -rf /", "sudo"},
		{"sudo rm -rf /", "rm_rf"},
		{"ls $(pwd)", "cmd_substitution"},
		{"IFS= read", "ifs_injection"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			desc := v.DescribeFailures(tt.command)
			if tt.contains != "No security issues detected" {
				assert.Contains(t, desc, tt.contains)
			} else {
				assert.Equal(t, "No security issues detected", desc)
			}
		})
	}
}

func TestBashValidator_SudoVariations(t *testing.T) {
	v := NewBashValidator()

	tests := []string{
		"sudo ls",
		"sudo rm file",
		"sudo -u root ls",
		"sudo -i",
		"sudo bash",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			failures := v.Validate(cmd)
			assert.True(t, len(failures) > 0)
			assert.Equal(t, "sudo", failures[0].Name)
		})
	}
}

func TestBashValidator_RmRfVariations(t *testing.T) {
	v := NewBashValidator()

	tests := []string{
		"rm -rf /",
		"rm -rf --no-preserve-root /",
		"rm -r -f /",
		"rm -rf /home",
		"rm -rf /var/log",
		"/bin/rm -rf /",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			failures := v.Validate(cmd)
			assert.True(t, len(failures) > 0)
			found := false
			for _, f := range failures {
				if f.Name == "rm_rf" {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected rm_rf to be detected for '%s'", cmd)
		})
	}
}

func TestBashValidator_SafeRmCommands(t *testing.T) {
	v := NewBashValidator()

	// These rm commands should NOT trigger rm_rf
	tests := []string{
		"rm file.txt",
		"rm -f file.txt",
		"rm *.go",
		"rm -i file",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			failures := v.Validate(cmd)
			for _, f := range failures {
				assert.NotEqual(t, "rm_rf", f.Name, "rm_rf should NOT be detected for '%s'", cmd)
			}
		})
	}
}

func TestBashValidator_CmdSubstitution(t *testing.T) {
	v := NewBashValidator()

	tests := []string{
		"ls $(pwd)",
		"echo $(whoami)",
		"cat $(find . -name '*.go')",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			failures := v.Validate(cmd)
			found := false
			for _, f := range failures {
				if f.Name == "cmd_substitution" {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected cmd_substitution for '%s'", cmd)
		})
	}
}

func TestBashValidator_IFSInjection(t *testing.T) {
	v := NewBashValidator()

	tests := []string{
		"IFS= read -r line",
		"IFS=':' read -ra arr",
		"IFS=,",
	}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			failures := v.Validate(cmd)
			found := false
			for _, f := range failures {
				if f.Name == "ifs_injection" {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected ifs_injection for '%s'", cmd)
		})
	}
}

func TestBashValidator_ComplexCommands(t *testing.T) {
	v := NewBashValidator()

	tests := []struct {
		command  string
		expected []string
	}{
		{"sudo rm -rf / && echo done", []string{"sudo", "rm_rf"}},
		{"ls | grep test | wc -l", nil},
		{"cat $(find . -type f) | grep pattern", []string{"cmd_substitution"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			failures := v.Validate(tt.command)
			var failureNames []string
			for _, f := range failures {
				failureNames = append(failureNames, f.Name)
			}
			assert.Equal(t, tt.expected, failureNames)
		})
	}
}
