package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LLMConfig struct {
	APIKey    string   `json:"api_key"`
	BaseURL   string   `json:"base_url"`
	Model     string   `json:"model"`
	Models    []string `json:"models"`
	UpdatedAt string   `json:"updated_at"`
}

type LLMConfigManager struct {
	mu         sync.RWMutex
	config     *LLMConfig
	configPath string
}

func NewLLMConfigManager(projectRoot, defaultAPIKey, defaultBaseURL, defaultModel string) *LLMConfigManager {
	configDir := filepath.Join(projectRoot, ".config")
	configPath := filepath.Join(configDir, "llm.json")

	mgr := &LLMConfigManager{
		configPath: configPath,
	}

	cfg := mgr.loadFromFile()
	if cfg == nil {
		cfg = &LLMConfig{
			APIKey:    defaultAPIKey,
			BaseURL:   defaultBaseURL,
			Model:     defaultModel,
			Models:    []string{defaultModel},
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
	}

	mgr.config = cfg
	return mgr
}

func (m *LLMConfigManager) loadFromFile() *LLMConfig {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil
	}

	var cfg LLMConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	return &cfg
}

func (m *LLMConfigManager) saveToFile() error {
	m.mu.RLock()
	cfg := &LLMConfig{
		APIKey:    m.config.APIKey,
		BaseURL:   m.config.BaseURL,
		Model:     m.config.Model,
		Models:    append([]string(nil), m.config.Models...),
		UpdatedAt: m.config.UpdatedAt,
	}
	configPath := m.configPath
	m.mu.RUnlock()

	return m.saveToFileUnsafe(cfg, configPath)
}

func (m *LLMConfigManager) Get() *LLMConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &LLMConfig{
		APIKey:    m.config.APIKey,
		BaseURL:   m.config.BaseURL,
		Model:     m.config.Model,
		Models:    append([]string(nil), m.config.Models...),
		UpdatedAt: m.config.UpdatedAt,
	}
}

func (m *LLMConfigManager) Update(apiKey, baseURL, model string, models []string) error {
	m.mu.Lock()
	if apiKey != "" {
		m.config.APIKey = apiKey
	}
	if baseURL != "" {
		m.config.BaseURL = baseURL
	}
	if model != "" {
		m.config.Model = model
	}
	if models != nil {
		m.config.Models = append([]string(nil), models...)
	}
	m.config.UpdatedAt = time.Now().Format(time.RFC3339)

	cfg := &LLMConfig{
		APIKey:    m.config.APIKey,
		BaseURL:   m.config.BaseURL,
		Model:     m.config.Model,
		Models:    append([]string(nil), m.config.Models...),
		UpdatedAt: m.config.UpdatedAt,
	}
	configPath := m.configPath
	m.mu.Unlock()

	return m.saveToFileUnsafe(cfg, configPath)
}

func (m *LLMConfigManager) saveToFileUnsafe(cfg *LLMConfig, configPath string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (m *LLMConfigManager) MaskAPIKey() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.config == nil {
		return ""
	}
	key := m.config.APIKey
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func (m *LLMConfigManager) HasAPIKey() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config == nil {
		return false
	}
	return m.config.APIKey != ""
}
