package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	Model            string
	WorkDir          string
	APIKey           string
	BaseURL          string
	Timeout          int
	MaxTokens        int
	ContextThreshold int
	BashTimeout      int
}

func LoadConfig() (*Config, error) {
	loadDotEnv()

	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DASHSCOPE_API_KEY environment variable is not set")
	}

	workDir := getWorkingDir()

	return &Config{
		Model:            "glm-5",
		WorkDir:          workDir,
		APIKey:           apiKey,
		BaseURL:          "https://coding.dashscope.aliyuncs.com/v1",
		Timeout:          120,
		MaxTokens:        8192,
		ContextThreshold: 50000,
		BashTimeout:      120,
	}, nil
}

func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for i := 0; i < 10; i++ {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			godotenv.Overload(envPath)
			return
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

func getWorkingDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
