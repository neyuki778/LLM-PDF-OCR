package llm

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Provider  string // "gemini" or "mineru"
	APIKey    string
	BaseURL   string // Optional, MinerU API 地址
	Model     string // Optional, 如 "gemini-3-flash-preview"
	PublicURL string // Optional, 本服务公开地址，将PDF暴露给LLM API提供商 需要
}

// LoadConfigFromEnv 从环境变量加载配置
func LoadConfigFromEnv() (Config, error) {
	provider := strings.TrimSpace(os.Getenv("LLM_PROVIDER"))
	if provider == "" {
		provider = "gemini" // 默认使用 gemini
	}
	provider = strings.ToLower(provider)

	cfg := Config{
		Provider:  provider,
		PublicURL: os.Getenv("PUBLIC_URL"),
	}

	switch provider {
	case "gemini":
		cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		cfg.Model = os.Getenv("GEMINI_MODEL")
		if cfg.Model == "" {
			cfg.Model = "gemini-3-flash-preview"
		}
		if cfg.APIKey == "" {
			return Config{}, fmt.Errorf("missing GEMINI_API_KEY for provider=gemini")
		}
	case "mineru":
		cfg.APIKey = os.Getenv("MINERU_TOKEN")
		cfg.BaseURL = os.Getenv("MINERU_BASE_URL")
		if cfg.APIKey == "" {
			return Config{}, fmt.Errorf("missing MINERU_TOKEN for provider=mineru")
		}
		if strings.TrimSpace(cfg.PublicURL) == "" {
			return Config{}, fmt.Errorf("missing PUBLIC_URL for provider=mineru")
		}
	default:
		return Config{}, fmt.Errorf("unknown LLM_PROVIDER: %s", provider)
	}

	return cfg, nil
}
