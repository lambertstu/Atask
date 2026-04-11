package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateTokens(t *testing.T) {
	// EstimateTokens uses len/4 approximation
	tests := []struct {
		name    string
		input   interface{}
		minimum int
	}{
		{"simple string", "hello world", 3},
		{"empty string", "", 0},
		{"slice of strings", []string{"a", "b", "c"}, 3},
		{"map", map[string]interface{}{"key": "value"}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			assert.GreaterOrEqual(t, result, tt.minimum)
		})
	}
}

func TestParseFloatToInt(t *testing.T) {
	tests := []struct {
		input    float64
		expected int
	}{
		{1.0, 1},
		{1.5, 1},
		{1.9, 1},
		{0.0, 0},
		{-1.5, -1},
		{100.99, 100},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.expected, ParseFloatToInt(tt.input))
		})
	}
}

func TestGetStringFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{"existing string", map[string]interface{}{"path": "test.go"}, "path", "test.go"},
		{"non-existing key", map[string]interface{}{"path": "test.go"}, "file", ""},
		{"empty map", map[string]interface{}{}, "path", ""},
		{"wrong type - int", map[string]interface{}{"path": 123}, "path", ""},
		{"wrong type - bool", map[string]interface{}{"path": true}, "path", ""},
		{"empty string", map[string]interface{}{"path": ""}, "path", ""},
		{"nil map", nil, "path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected int
	}{
		{"existing int as float64", map[string]interface{}{"limit": float64(10)}, "limit", 10},
		{"non-existing key", map[string]interface{}{"limit": float64(10)}, "count", 0},
		{"empty map", map[string]interface{}{}, "limit", 0},
		{"wrong type - string", map[string]interface{}{"limit": "10"}, "limit", 0},
		{"wrong type - int", map[string]interface{}{"limit": 10}, "limit", 0},
		{"zero value", map[string]interface{}{"limit": float64(0)}, "limit", 0},
		{"negative value", map[string]interface{}{"limit": float64(-5)}, "limit", -5},
		{"nil map", nil, "limit", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIntFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBoolFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected bool
	}{
		{"existing true", map[string]interface{}{"flag": true}, "flag", true},
		{"existing false", map[string]interface{}{"flag": false}, "flag", false},
		{"non-existing key", map[string]interface{}{"flag": true}, "other", false},
		{"empty map", map[string]interface{}{}, "flag", false},
		{"wrong type - string", map[string]interface{}{"flag": "true"}, "flag", false},
		{"wrong type - int", map[string]interface{}{"flag": 1}, "flag", false},
		{"nil map", nil, "flag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBoolFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSliceFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected []interface{}
	}{
		{"existing slice", map[string]interface{}{"items": []interface{}{"a", "b", "c"}}, "items", []interface{}{"a", "b", "c"}},
		{"non-existing key", map[string]interface{}{"items": []interface{}{"a"}}, "other", nil},
		{"empty map", map[string]interface{}{}, "items", nil},
		{"wrong type - string", map[string]interface{}{"items": "a,b,c"}, "items", nil},
		{"empty slice", map[string]interface{}{"items": []interface{}{}}, "items", []interface{}{}},
		{"nil map", nil, "items", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSliceFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetMapFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected map[string]interface{}
	}{
		{"existing map", map[string]interface{}{
			"nested": map[string]interface{}{"a": 1, "b": 2},
		}, "nested", map[string]interface{}{"a": 1, "b": 2}},
		{"non-existing key", map[string]interface{}{
			"nested": map[string]interface{}{"a": 1},
		}, "other", nil},
		{"empty map", map[string]interface{}{}, "nested", nil},
		{"wrong type - string", map[string]interface{}{"nested": "{a:1}"}, "nested", nil},
		{"empty nested map", map[string]interface{}{
			"nested": map[string]interface{}{},
		}, "nested", map[string]interface{}{}},
		{"nil map", nil, "nested", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMapFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntArrayFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected []int
	}{
		{"existing []int", map[string]interface{}{"nums": []int{1, 2, 3}}, "nums", []int{1, 2, 3}},
		{"existing []interface{} with float64", map[string]interface{}{
			"nums": []interface{}{float64(1), float64(2), float64(3)},
		}, "nums", []int{1, 2, 3}},
		{"existing []interface{} with int", map[string]interface{}{
			"nums": []interface{}{1, 2, 3},
		}, "nums", []int{1, 2, 3}},
		{"mixed types", map[string]interface{}{
			"nums": []interface{}{float64(1), 2, float64(3)},
		}, "nums", []int{1, 2, 3}},
		{"non-existing key", map[string]interface{}{"nums": []int{1}}, "other", nil},
		{"empty map", map[string]interface{}{}, "nums", nil},
		{"wrong type - string", map[string]interface{}{"nums": "1,2,3"}, "nums", nil},
		{"empty slice", map[string]interface{}{"nums": []interface{}{}}, "nums", nil},
		{"nil map", nil, "nums", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetIntArrayFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringSliceFromMap(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected []string
	}{
		{"existing []string", map[string]interface{}{"paths": []string{"a", "b", "c"}}, "paths", []string{"a", "b", "c"}},
		{"existing []interface{} with strings", map[string]interface{}{
			"paths": []interface{}{"a", "b", "c"},
		}, "paths", []string{"a", "b", "c"}},
		{"non-existing key", map[string]interface{}{"paths": []string{"a"}}, "other", nil},
		{"empty map", map[string]interface{}{}, "paths", nil},
		{"wrong type - int", map[string]interface{}{"paths": 123}, "paths", nil},
		{"mixed types - some non-string", map[string]interface{}{
			"paths": []interface{}{"a", 123, "c"},
		}, "paths", []string{"a", "c"}},
		{"empty slice", map[string]interface{}{"paths": []interface{}{}}, "paths", nil},
		{"nil map", nil, "paths", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetStringSliceFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
