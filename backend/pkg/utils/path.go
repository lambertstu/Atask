package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

type PathEscapeError struct {
	Path string
}

func (e *PathEscapeError) Error() string {
	return fmt.Sprintf("path escapes workspace: %s", e.Path)
}

func SafePath(baseDir string, allowedDirs []string, p string) (string, error) {
	var absPath string
	var err error

	if filepath.IsAbs(p) {
		absPath, err = filepath.Abs(p)
	} else {
		absPath, err = filepath.Abs(filepath.Join(baseDir, p))
	}

	if err != nil {
		return "", err
	}

	// 优先检查 allowedDirs（支持跨项目授权）
	for _, allowed := range allowedDirs {
		if strings.HasPrefix(absPath, allowed) {
			return absPath, nil
		}
	}

	// 再检查 baseDir（默认工作目录）
	if strings.HasPrefix(absPath, baseDir) {
		return absPath, nil
	}

	return absPath, &PathEscapeError{Path: absPath}
}

func MatchGlob(s, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(s, prefix)
	}
	return s == pattern
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ContainsInt(ints []int, i int) bool {
	for _, v := range ints {
		if v == i {
			return true
		}
	}
	return false
}

func UniqueInts(ints []int) []int {
	seen := make(map[int]bool)
	var result []int
	for _, i := range ints {
		if !seen[i] {
			seen[i] = true
			result = append(result, i)
		}
	}
	return result
}

func RemoveInt(ints []int, i int) []int {
	var result []int
	for _, v := range ints {
		if v != i {
			result = append(result, v)
		}
	}
	return result
}
