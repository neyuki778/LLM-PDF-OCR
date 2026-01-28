package llm

import "os"

type Config struct {
	Provider  string // "gemini" or "mineru"
	APIKey    string
	BaseURL   string // Optional, MinerU API 地址
	Model     string // Optional, 如 "gemini-3-flash-preview"
	PublicURL string // Optional, 本服务公开地址，将PDF暴露给LLM API提供商 需要
}

// LoadConfigFromEnv 从环境变量加载配置
func LoadConfigFromEnv() Config {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "gemini" // 默认使用 gemini
	}

	cfg := Config{
		Provider:  provider,
		PublicURL: os.Getenv("PUBLIC_URL"),
	}

	switch provider {
	case "gemini":
		cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		cfg.Model = os.Getenv("GEMINI_MODEL")
	case "mineru":
		cfg.APIKey = os.Getenv("MINERU_TOKEN")
		cfg.BaseURL = os.Getenv("MINERU_BASE_URL")
	}

	return cfg
}
