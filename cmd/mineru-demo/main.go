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

	client := mineru.NewClient(baseURL, token)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.CreateTask(ctx, mineru.CreateTaskRequest{
		URL:          fileURL,
		ModelVersion: modelVersion,
	})
	if err != nil {
		log.Fatalf("create task failed: %v", err)
	}

	fmt.Printf("task_id=%s\n", resp.Data.TaskID)
}
