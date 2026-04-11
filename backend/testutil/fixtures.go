package testutil

var SampleTaskJSON = `{"id": "test-1", "title": "Test Task", "status": "pending", "priority": "high"}`

var SampleHookJSON = `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "write_file",
        "command": "echo 'blocked' && exit 1"
      }
    ],
    "PostToolUse": [
      {
        "matcher": "read_file",
        "command": "echo '{\"additionalContext\": \"file read completed\"}'"
      }
    ]
  }
}`

var SampleMemoryMD = `---
type: knowledge
description: Test memory
---
This is test memory content.

Key points:
- Point 1
- Point 2
`

var SampleSkillMD = `---
name: test-skill
description: Test skill for demonstration
---
# Test Skill

This skill demonstrates basic functionality.

## Usage
Use this skill for testing purposes.
`

var SampleGoCode = `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
