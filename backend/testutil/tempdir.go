package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

type TempDir struct {
	Path string
	t    *testing.T
}

func NewTempDir(t *testing.T) *TempDir {
	path, err := os.MkdirTemp("", "agent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return &TempDir{Path: path, t: t}
}

func (td *TempDir) CreateFile(name, content string) string {
	fullPath := filepath.Join(td.Path, name)
	dir := filepath.Dir(fullPath)
	if dir != td.Path {
		os.MkdirAll(dir, 0755)
	}
	err := os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		td.t.Fatalf("Failed to create file %s: %v", name, err)
	}
	return fullPath
}

func (td *TempDir) ReadFile(name string) string {
	fullPath := filepath.Join(td.Path, name)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		td.t.Fatalf("Failed to read file %s: %v", name, err)
	}
	return string(content)
}

func (td *TempDir) Exists(name string) bool {
	fullPath := filepath.Join(td.Path, name)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (td *TempDir) Cleanup() {
	os.RemoveAll(td.Path)
}

func (td *TempDir) Subdir(name string) *TempDir {
	subPath := filepath.Join(td.Path, name)
	os.MkdirAll(subPath, 0755)
	return &TempDir{Path: subPath, t: td.t}
}
