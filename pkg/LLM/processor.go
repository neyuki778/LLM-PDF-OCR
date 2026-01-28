package llm

import (
	"context"
	"fmt"

	mineru "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM/MinerU"
	gemini "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM/gemini"
)

type PDFProcessor interface {
	ProcessPDF(ctx context.Context, pdfPath string) (string, error)
}

type Config struct {
	Provider  string // "gemini" or "mineru"
	APIKey    string
	BaseURL   string // Optional, MinerU API 地址
	Model     string // Optional, 如 "gemini-3-flash-preview"
	PublicURL string // Optional, 本服务公开地址，将PDF暴露给LLM API提供商 需要
}

func NewProcessor(cfg Config) (PDFProcessor, error) {
	switch cfg.Provider {
	case "gemini":
		return gemini.NewClient(cfg.APIKey, cfg.Model)
	case "mineru":
		return mineru.NewClient(cfg.BaseURL, cfg.APIKey, cfg.PublicURL), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}