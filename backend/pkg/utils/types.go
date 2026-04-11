package utils

import (
	"encoding/json"
)

func EstimateTokens(messages interface{}) int {
	data, _ := json.Marshal(messages)
	return len(string(data)) / 4
}

func ParseFloatToInt(f float64) int {
	return int(f)
}

func GetStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func GetIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

func GetBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

func GetSliceFromMap(m map[string]interface{}, key string) []interface{} {
	if val, ok := m[key].([]interface{}); ok {
		return val
	}
	return nil
}

func GetMapFromMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}

func GetIntArrayFromMap(m map[string]interface{}, key string) []int {
	if s, ok := m[key].([]int); ok {
		return s
	}

	val := GetSliceFromMap(m, key)
	if val == nil {
		return nil
	}
	var result []int
	for _, item := range val {
		if f, ok := item.(float64); ok {
			result = append(result, int(f))
		} else if i, ok := item.(int); ok {
			result = append(result, i)
		}
	}
	return result
}

func GetStringSliceFromMap(m map[string]interface{}, key string) []string {
	if s, ok := m[key].([]string); ok {
		return s
	}

	val := GetSliceFromMap(m, key)
	if val == nil {
		return nil
	}
	var result []string
	for _, item := range val {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
