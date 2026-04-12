package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-base/internal/config"
	"agent-base/internal/llm"
	"agent-base/internal/session"
	"agent-base/internal/tools"
	"agent-base/internal/tools/builtin"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	llmClient := llm.NewClient(cfg)

	globalRegistry := tools.NewDefaultRegistry()
	globalRegistry.Register(builtin.NewBashTool(cfg.WorkDir, cfg.BashTimeout))
	globalRegistry.Register(builtin.NewReadFileTool(cfg.WorkDir))
	globalRegistry.Register(builtin.NewWriteFileTool(cfg.WorkDir))
	globalRegistry.Register(builtin.NewEditFileTool(cfg.WorkDir))
	globalRegistry.Register(builtin.NewWebFetchTool())

	sessionMgr := session.NewSessionManager(cfg, llmClient, globalRegistry)

	sessionMgr.RestoreSessions(cfg.WorkDir)

	defaultSession, err := sessionMgr.NewSession(cfg.WorkDir, "default")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating default session: %v\n", err)
		os.Exit(1)
	}
	currentSessionID := defaultSession.ID

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
		}
	}()

	fmt.Println("\033[32m[Agent Ready - Session Management Enabled]\033[0m")
	fmt.Println("Commands: /new <name>, /sessions, /switch <id>, /send <id> <msg>, /start <id>, /pause <id>, /cancel <id>")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\033[36magent [%s] >> \033[0m", currentSessionID)
		query, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		query = strings.TrimSpace(query)
		if query == "" || strings.ToLower(query) == "q" || strings.ToLower(query) == "exit" {
			break
		}

		if handleSystemCommand(query, sessionMgr, &currentSessionID) {
			continue
		}

		if strings.HasPrefix(query, "/") {
			fmt.Println("Unknown command. Available commands: /new, /sessions, /switch, /send, /start, /pause, /cancel")
			continue
		}

		currentSession, ok := sessionMgr.GetSession(currentSessionID)
		if !ok {
			fmt.Printf("Error: session %s not found\n", currentSessionID)
			continue
		}

		if currentSession.GetStatus() == session.StatusHumanReview {
			fmt.Println("Session is waiting for permission approval. Use /approve or /reject")
			continue
		}

		eventCh := sessionMgr.Subscribe(currentSessionID)
		defer sessionMgr.Unsubscribe(currentSessionID, eventCh)

		err = sessionMgr.SendMessage(currentSessionID, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		waitForResponse(ctx, eventCh)
	}

	sessionMgr.Shutdown()
}

func handleSystemCommand(query string, sessionMgr *session.SessionManager, currentSessionID *string) bool {
	if strings.HasPrefix(query, "/new ") {
		name := strings.TrimSpace(strings.TrimPrefix(query, "/new "))
		if name == "" {
			name = fmt.Sprintf("session_%d", time.Now().Unix())
		}

		cfg, _ := config.LoadConfig()
		newSession, err := sessionMgr.NewSession(cfg.WorkDir, name)
		if err != nil {
			fmt.Printf("Error creating session: %v\n", err)
			return true
		}

		*currentSessionID = newSession.ID
		fmt.Printf("Created session: %s\n", newSession.GetInfo())
		return true
	}

	if query == "/sessions" {
		sessions := sessionMgr.ListSessions()
		if len(sessions) == 0 {
			fmt.Println("No sessions.")
		} else {
			for _, s := range sessions {
				fmt.Println(s.String())
			}
		}
		return true
	}

	if strings.HasPrefix(query, "/switch ") {
		sessionID := strings.TrimSpace(strings.TrimPrefix(query, "/switch "))
		if _, ok := sessionMgr.GetSession(sessionID); ok {
			*currentSessionID = sessionID
			fmt.Printf("Switched to session: %s\n", sessionID)
		} else {
			fmt.Printf("Session %s not found\n", sessionID)
		}
		return true
	}

	if strings.HasPrefix(query, "/send ") {
		parts := strings.SplitN(strings.TrimPrefix(query, "/send "), " ", 3)
		if len(parts) < 2 {
			fmt.Println("Usage: /send <session_id> <message>")
			return true
		}

		sessionID := parts[0]
		message := parts[1]
		if len(parts) > 2 {
			message = parts[1] + " " + parts[2]
		}

		eventCh := sessionMgr.Subscribe(sessionID)
		defer sessionMgr.Unsubscribe(sessionID, eventCh)

		err := sessionMgr.SendMessage(sessionID, message)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return true
		}

		waitForResponse(context.Background(), eventCh)
		return true
	}

	if strings.HasPrefix(query, "/start ") {
		sessionID := strings.TrimSpace(strings.TrimPrefix(query, "/start "))
		err := sessionMgr.SendControl(sessionID, "start")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Started session: %s\n", sessionID)
		}
		return true
	}

	if strings.HasPrefix(query, "/pause ") {
		sessionID := strings.TrimSpace(strings.TrimPrefix(query, "/pause "))
		err := sessionMgr.SendControl(sessionID, "pause")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Paused session: %s\n", sessionID)
		}
		return true
	}

	if strings.HasPrefix(query, "/cancel ") {
		sessionID := strings.TrimSpace(strings.TrimPrefix(query, "/cancel "))
		err := sessionMgr.StopSession(sessionID)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Cancelled session: %s\n", sessionID)
		}
		return true
	}

	if strings.HasPrefix(query, "/approve ") {
		parts := strings.Split(strings.TrimPrefix(query, "/approve "), " ")
		if len(parts) < 2 {
			fmt.Println("Usage: /approve <session_id> <request_id>")
			return true
		}
		sessionID := parts[0]
		requestID := parts[1]
		err := sessionMgr.SendPermissionResponse(sessionID, requestID, true)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Approved permission request %s in session %s\n", requestID, sessionID)
		}
		return true
	}

	if strings.HasPrefix(query, "/reject ") {
		parts := strings.Split(strings.TrimPrefix(query, "/reject "), " ")
		if len(parts) < 2 {
			fmt.Println("Usage: /reject <session_id> <request_id>")
			return true
		}
		sessionID := parts[0]
		requestID := parts[1]
		err := sessionMgr.SendPermissionResponse(sessionID, requestID, false)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Rejected permission request %s in session %s\n", requestID, sessionID)
		}
		return true
	}

	return false
}

func waitForResponse(ctx context.Context, eventCh chan session.SessionEvent) {
	for {
		select {
		case event := <-eventCh:
			switch event.Type {
			case session.EventStatusChange:
				from, _ := event.Data["from"].(string)
				to, _ := event.Data["to"].(string)
				fmt.Printf("\033[34m[Status: %s -> %s]\033[0m\n", from, to)
				if to == "completed" {
					return
				}
			case session.EventOutput:
				content, _ := event.Data["content"].(string)
				isFinal, _ := event.Data["is_final"].(bool)
				fmt.Println(content)
				if isFinal {
					return
				}
			case session.EventToolCall:
				toolName, _ := event.Data["tool_name"].(string)
				fmt.Printf("\033[33m[Tool: %s]\033[0m\n", toolName)
			case session.EventPermission:
				requestID, _ := event.Data["request_id"].(string)
				toolName, _ := event.Data["tool_name"].(string)
				reason, _ := event.Data["reason"].(string)
				fmt.Printf("\033[31m[Permission Request %s] Tool: %s, Reason: %s\033[0m\n", requestID, toolName, reason)
				fmt.Println("Use /approve or /reject <session_id> <request_id> to respond")
			case session.EventError:
				errMsg, _ := event.Data["error"].(string)
				fmt.Printf("\033[31m[Error: %s]\033[0m\n", errMsg)
				return
			}
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			fmt.Println("[Timeout waiting for response]")
			return
		}
	}
}
