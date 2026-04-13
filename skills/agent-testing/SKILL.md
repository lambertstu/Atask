---
name: agent-testing
description: Guides the execution of the Agent's automated interactive tests and the writing of expect test scripts. Use when the user asks to test the agent, write agent tests, modify test scenarios, or mentions agent_test.exp.
---

# Agent Automated Interactive Testing

This skill guides you on how to run and write automated conversational tests for the Agent using `expect`.

## Core Workflow

### 1. Running Existing Tests
Before executing tests, ensure that `expect` and Go 1.23+ are installed in your environment.

```bash
cd backend
# Run standard test
./test/agent_test.exp

# Or run with detailed log output for troubleshooting
script -c "./test/agent_test.exp" test_output.log
```

### 2. Writing/Modifying Test Scripts
When writing `expect` test cases, **the most critical part is handling the Agent's dynamic permission confirmations**. Please follow this standard template:

```expect
#!/usr/bin/expect -f

set timeout 60
log_user 1

# 1. Start the Agent
spawn go run main.go
expect "agent >> "

# 2. Send test command
send "Read the contents of the main.go file\r"

# 3. Handle response (⚠️ Must include the permission confirmation branch)
expect {
    # Scenario A: Agent successfully responds and waits for next input
    "agent >> " {}
    
    # Scenario B: Tool call triggers security interception requiring user confirmation
    "(y/N):" { 
        send "y\r"
        exp_continue
    }
    
    # Scenario C: Response timeout
    timeout { 
        puts "ERROR: Timeout waiting for agent response"
        exit 1 
    }
}

# 4. End test
send "exit\r"
expect eof
```

## Test Scenario Examples

**Testing built-in system commands:**
For pure system commands that don't require tool call confirmation, you can use simple expect statements:
```expect
send "/mode plan\r"
expect "agent >> "

send "/tasks\r"
expect "agent >> "
```

**Testing complex tool calls (like subagents or Bash):**
```expect
send "Use a subagent to analyze the project structure\r"
expect {
    "agent >> " {}
    "(y/N):" { send "y\r"; exp_continue }
}
```

## Validation & Checkpoints

A complete test execution should cover the following key validation points:
1. **Normal Startup**: The Agent starts successfully and displays the version or welcome message (e.g., `[Agent MVP Ready - Refactored Architecture]`).
2. **Tool Triggering**: Expected tools (`read_file`, `bash`, `task_create`, etc.) are successfully invoked without being permanently blocked by permissions.
3. **Safe Exit**: After sending `exit` at the end of the test, the process terminates normally.

## Maintenance & Cleanup

Tests generate local temporary state data. It is recommended to perform cleanup before or after each new test run to ensure test idempotency:
```bash
# Execute in the project root directory
rm -rf backend/.tasks/* backend/.memory/* backend/.runtime-tasks/*
```
