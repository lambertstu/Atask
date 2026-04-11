package utils

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
