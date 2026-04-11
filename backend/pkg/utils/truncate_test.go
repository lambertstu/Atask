package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"exactly max length", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello"},
		{"empty string", "", 10, ""},
		{"zero maxLen", "hello", 0, ""},
		{"unicode characters - byte truncation", "你好世界", 3, "你"},
		{"mixed ascii and unicode - byte truncation", "hello你好", 8, "hello你"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate_UnicodeByteLimitation(t *testing.T) {
	// Note: Truncate works on bytes, not characters
	// A Chinese character is 3 bytes in UTF-8
	longUnicode := "你好世界测试"
	result := Truncate(longUnicode, 6) // 6 bytes = 2 Chinese characters
	assert.Len(t, result, 6)
	assert.Equal(t, "你好", result)
}

func TestTruncate_PreservesOriginal(t *testing.T) {
	original := "important content"
	result := Truncate(original, 100)
	assert.Equal(t, original, result)
	assert.Equal(t, "important content", original)
}

func TestTruncate_DoesNotAddEllipsis(t *testing.T) {
	longString := "this is a very long string that should be truncated"
	result := Truncate(longString, 10)
	assert.Equal(t, "this is a ", result)
	assert.NotContains(t, result, "...")
}

func TestMin(t *testing.T) {
	tests := []struct {
		a        int
		b        int
		expected int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 10, 0},
		{-5, 5, -5},
		{-10, -5, -10},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.expected, Min(tt.a, tt.b))
		})
	}
}
