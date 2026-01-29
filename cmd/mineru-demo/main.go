package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	mineru "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM/MinerU"
)

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func main() {
	baseURL := getenv("MINERU_BASE_URL", "https://mineru.net")
	token := os.Getenv("MINERU_TOKEN")
	fileURL := os.Getenv("MINERU_FILE_URL")
	modelVersion := getenv("MINERU_MODEL_VERSION", "vlm")

	if token == "" || fileURL == "" {
		log.Fatalf("missing env vars: MINERU_TOKEN=%t MINERU_FILE_URL=%t", token != "", fileURL != "")
	}

	client := mineru.NewClient(baseURL, token, baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.CreateTask(ctx, mineru.CreateTaskRequest{
		URL:          fileURL,
		ModelVersion: modelVersion,
	})
	if err != nil {
		log.Fatalf("create task failed: %v", err)
	}

	fmt.Printf("Task created: %s\n", resp.Data.TaskID)
	fmt.Println("Waiting for completion and extracting content...")

	// 使用一站式方法：等待 + 下载 + 提取
	content, err := client.ProcessTask(ctx, resp.Data.TaskID)
	if err != nil {
		log.Fatalf("process task failed: %v", err)
	}

	fmt.Println("\n=== Extraction Complete ===")
	fmt.Printf("Markdown length: %d bytes\n", len(content.Markdown))
	fmt.Printf("Layout JSON length: %d bytes\n", len(content.LayoutJSON))
	fmt.Printf("Source PDF length: %d bytes\n", len(content.SourcePDF))

	// 打印 Markdown 内容预览（前 500 字符）
	fmt.Println("\n=== Markdown Preview ===")
	preview := content.Markdown
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	fmt.Println(preview)
}
