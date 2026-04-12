package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"agent-base/internal/api"
	"agent-base/internal/config"
	"agent-base/internal/llm"
	"agent-base/internal/session"
	"agent-base/internal/tools"
	"agent-base/internal/tools/builtin"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	llmClient := llm.NewClient(cfg)

	globalRegistry := tools.NewDefaultRegistry()
	globalRegistry.Register(builtin.NewBashTool(cfg.WorkDir, cfg.BashTimeout))
	globalRegistry.Register(builtin.NewReadFileTool(cfg.WorkDir))
	globalRegistry.Register(builtin.NewWriteFileTool(cfg.WorkDir))
	globalRegistry.Register(builtin.NewEditFileTool(cfg.WorkDir))
	globalRegistry.Register(builtin.NewWebFetchTool())

	sessionMgr := session.NewSessionManager(cfg, llmClient, globalRegistry)

	if err := sessionMgr.RestoreSessions(cfg.WorkDir); err != nil {
		log.Printf("Warning: Failed to restore sessions: %v", err)
	}

	apiServer := api.NewAPIServer(sessionMgr, cfg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 Atask Web Server starting on port %s...\n", port)
	fmt.Printf("📡 WebSocket endpoint: ws://localhost:%s/api/ws\n", port)
	fmt.Printf("🌐 API endpoint: http://localhost:%s/api\n", port)

	go func() {
		if err := http.ListenAndServe(":"+port, apiServer.Mux()); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")
	sessionMgr.Shutdown()
	fmt.Println("✅ Server stopped")
}
