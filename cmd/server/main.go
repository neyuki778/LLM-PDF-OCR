package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/neyuki778/LLM-PDF-OCR/internal/api"
	"github.com/neyuki778/LLM-PDF-OCR/internal/task"
	llm "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM"
)

func main() {
	godotenv.Load()

	// 初始化 TaskManager（3 个 worker）
	config := llm.Config{}
	tm, err := task.NewTaskManager(3, config)
	if err != nil {
		log.Fatalf("Failed to create TaskManager: %v", err)
	}
	if err := tm.Start(); err != nil {
		log.Fatalf("Failed to start TaskManager: %v", err)
	}
	defer tm.ShutDown()

	// 创建并启动 HTTP 服务
	server := api.NewServer(tm)
	log.Println("Server starting on :8080")
	if err := server.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
