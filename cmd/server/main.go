package main

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	api "github.com/neyuki778/LLM-PDF-OCR/internal/api"
	redis "github.com/neyuki778/LLM-PDF-OCR/internal/store/redis"
	task "github.com/neyuki778/LLM-PDF-OCR/internal/task"
	llm "github.com/neyuki778/LLM-PDF-OCR/pkg/LLM"
)

func main() {
	godotenv.Load()

	// 从环境变量加载 LLM 配置
	config, err := llm.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	addr := os.Getenv("REDIS_ADDRESS")
	if addr == "" {
		addr = "localhost:6379"
	}
	r, err := redis.NewClient(addr)
	if err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}
	rs := redis.NewRedisStore(r, 5 * time.Hour)

	tm, err := task.NewTaskManager(3, config, rs)
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
