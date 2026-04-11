package utils

import (
	"path/filepath"
	"testing"

	"agent-base/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafePath_WithinWorkspace(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"relative path", "test.go", filepath.Join(tempDir.Path, "test.go")},
		{"nested relative path", "internal/engine/test.go", filepath.Join(tempDir.Path, "internal/engine/test.go")},
		{"absolute path inside", filepath.Join(tempDir.Path, "test.go"), filepath.Join(tempDir.Path, "test.go")},
		{"absolute nested path", filepath.Join(tempDir.Path, "pkg/utils/test.go"), filepath.Join(tempDir.Path, "pkg/utils/test.go")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafePath(tempDir.Path, nil, tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafePath_PathEscape(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../test.go"},
		{"absolute outside", "/etc/passwd"},
		{"sibling directory", filepath.Join(filepath.Dir(tempDir.Path), "other-project/test.go")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafePath(tempDir.Path, nil, tt.path)
			require.Error(t, err)
			assert.IsType(t, &PathEscapeError{}, err)
			escapeErr := err.(*PathEscapeError)
			assert.Equal(t, result, escapeErr.Path)
		})
	}
}

func TestSafePath_AllowedDirs(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	otherDir := testutil.NewTempDir(t)
	defer otherDir.Cleanup()

	// Use absolute paths that are definitely outside workspace
	outsidePath := "/etc/testoutside.go"

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		expectError bool
	}{
		{"allowed dir - single", filepath.Join(otherDir.Path, "test.go"), []string{otherDir.Path}, false},
		{"allowed dir - multiple", filepath.Join(otherDir.Path, "nested/test.go"), []string{"/tmp", otherDir.Path}, false},
		{"outside workspace - no allowed", outsidePath, []string{}, true},
		{"outside workspace - wrong allowed", outsidePath, []string{"/var"}, true},
		{"workspace path - no allowed needed", filepath.Join(tempDir.Path, "test.go"), []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafePath(tempDir.Path, tt.allowedDirs, tt.path)
			if tt.expectError {
				require.Error(t, err)
				assert.IsType(t, &PathEscapeError{}, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestSafePath_EdgeCases(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	tests := []struct {
		name        string
		path        string
		baseDir     string
		allowedDirs []string
		expectError bool
	}{
		{"empty path resolves to baseDir", "", tempDir.Path, nil, false},
		{"dot path resolves to baseDir", ".", tempDir.Path, nil, false},
		{"double dot escapes", "..", tempDir.Path, nil, true},
		{"triple dot treated as literal", "...", tempDir.Path, nil, false},
		{"symlink like path normalized", "link/../test.go", tempDir.Path, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafePath(tt.baseDir, tt.allowedDirs, tt.path)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestSafePath_PriorityCheck(t *testing.T) {
	tempDir := testutil.NewTempDir(t)
	defer tempDir.Cleanup()

	otherDir := testutil.NewTempDir(t)
	defer otherDir.Cleanup()

	// Test that allowedDirs is checked BEFORE baseDir
	// This is the key fix we made earlier
	pathInOtherDir := filepath.Join(otherDir.Path, "test.go")
	allowedDirs := []string{otherDir.Path}

	result, err := SafePath(tempDir.Path, allowedDirs, pathInOtherDir)
	require.NoError(t, err)
	assert.Equal(t, pathInOtherDir, result)
}

func TestPathEscapeError_Error(t *testing.T) {
	err := &PathEscapeError{Path: "/etc/passwd"}
	assert.Equal(t, "path escapes workspace: /etc/passwd", err.Error())
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		match   bool
	}{
		{"anything", "*", true},
		{"test.go", "test*", true},
		{"test.go", "*.go", false},
		{"test.go", "test.go", true},
		{"test.go", "other.go", false},
		{"internal/test.go", "internal*", true},
		{"internal/test.go", "pkg*", false},
		{"", "*", true},
		{"", "", true},
		{"test", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.pattern, func(t *testing.T) {
			assert.Equal(t, tt.match, MatchGlob(tt.s, tt.pattern))
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice  []string
		item   string
		result bool
	}{
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{[]string{"a"}, "a", true},
		{[]string{"a", "a", "a"}, "a", true},
		{nil, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			assert.Equal(t, tt.result, Contains(tt.slice, tt.item))
		})
	}
}

func TestContainsInt(t *testing.T) {
	tests := []struct {
		ints   []int
		i      int
		result bool
	}{
		{[]int{1, 2, 3}, 2, true},
		{[]int{1, 2, 3}, 4, false},
		{[]int{}, 1, false},
		{[]int{5}, 5, true},
		{nil, 1, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.result, ContainsInt(tt.ints, tt.i))
		})
	}
}

func TestUniqueInts(t *testing.T) {
	tests := []struct {
		input    []int
		expected []int
	}{
		{[]int{1, 2, 3, 4}, []int{1, 2, 3, 4}},
		{[]int{1, 1, 2, 2}, []int{1, 2}},
		{[]int{1, 1, 1}, []int{1}},
		{[]int{}, nil},
		{[]int{5, 3, 5, 3, 1}, []int{5, 3, 1}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := UniqueInts(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveInt(t *testing.T) {
	tests := []struct {
		ints     []int
		remove   int
		expected []int
	}{
		{[]int{1, 2, 3}, 2, []int{1, 3}},
		{[]int{1, 2, 3}, 4, []int{1, 2, 3}},
		{[]int{1, 1, 1}, 1, nil},
		{[]int{}, 1, nil},
		{[]int{5, 5, 3}, 5, []int{3}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := RemoveInt(tt.ints, tt.remove)
			assert.Equal(t, tt.expected, result)
		})
	}
}
