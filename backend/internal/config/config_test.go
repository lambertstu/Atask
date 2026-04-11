package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_WithAPIKey(t *testing.T) {
	// Set env
	os.Setenv("DASHSCOPE_API_KEY", "test-key")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.APIKey)
}

func TestConfig_Defaults(t *testing.T) {
	os.Setenv("DASHSCOPE_API_KEY", "test-key")

	cfg, err := LoadConfig()
	assert.NoError(t, err)

	assert.Equal(t, 120, cfg.Timeout)
	assert.Equal(t, 8192, cfg.MaxTokens)
	assert.Equal(t, 50000, cfg.ContextThreshold)
	assert.Equal(t, 120, cfg.BashTimeout)
}

func TestConfig_Model(t *testing.T) {
	os.Setenv("DASHSCOPE_API_KEY", "test-key")

	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.Equal(t, "glm-5", cfg.Model)
}
