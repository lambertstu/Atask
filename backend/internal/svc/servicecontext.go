// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"agent-base/internal/config"
	"agent-base/internal/engine"
	"agent-base/internal/llm"
	"agent-base/internal/systems/project"
	"agent-base/internal/systems/session"
	"agent-base/pkg/events"
)

type ServiceContext struct {
	Config         config.Config
	SessionManager *session.SessionManager
	EngineManager  *engine.EngineManager
	EventBus       *events.EventBus
	ProjectManager *project.ProjectManager
	LLMConfigMgr   *config.LLMConfigManager
	LLMClient      *llm.DashScopeClient
}

func NewServiceContext(cfg config.Config) *ServiceContext {
	llmConfigMgr := config.NewLLMConfigManager(cfg.ProjectRoot, cfg.APIKey, cfg.BaseURL, cfg.Model)

	llmCfg := llmConfigMgr.Get()
	llmClient := llm.NewClient(llmCfg.APIKey, llmCfg.BaseURL, llmCfg.Model)

	eventBus := events.NewEventBus()

	sessionManager := session.NewSessionManager(cfg.ProjectRoot, eventBus)

	projectManager := project.NewProjectManager(cfg.ProjectRoot)

	engineManager := engine.NewEngineManager(cfg, llmClient)

	return &ServiceContext{
		Config:         cfg,
		SessionManager: sessionManager,
		EngineManager:  engineManager,
		EventBus:       eventBus,
		ProjectManager: projectManager,
		LLMConfigMgr:   llmConfigMgr,
		LLMClient:      llmClient,
	}
}
