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

func NewProcessor(cfg Config) (PDFProcessor, error) {
	switch cfg.Provider {
	case "gemini":
		return gemini.NewClient(cfg.APIKey, cfg.Model, cfg.PublicURL)
	case "mineru":
		return mineru.NewClient(cfg.BaseURL, cfg.APIKey, cfg.PublicURL), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
