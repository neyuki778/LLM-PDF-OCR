package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	api "github.com/neyuki778/LLM-PDF-OCR/internal/api"
	auth "github.com/neyuki778/LLM-PDF-OCR/internal/auth"
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
	rs := redis.NewRedisStore(r, 5*time.Hour)

	tm, err := task.NewTaskManager(3, config, rs)
	if err != nil {
		log.Fatalf("Failed to create TaskManager: %v", err)
	}
	if err := tm.Start(); err != nil {
		log.Fatalf("Failed to start TaskManager: %v", err)
	}
	defer tm.ShutDown()

	var authService *auth.Service
	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret != "" {
		sqlitePath := os.Getenv("SQLITE_PATH")
		if sqlitePath == "" {
			sqlitePath = "./data/app.db"
		}

		authStore, err := auth.NewSQLiteStore(sqlitePath)
		if err != nil {
			log.Fatalf("Failed to init auth sqlite store: %v", err)
		}
		defer authStore.Close()

		accessTTL, err := parseDurationEnv("JWT_ACCESS_TTL", 15*time.Minute)
		if err != nil {
			log.Fatalf("Invalid JWT_ACCESS_TTL: %v", err)
		}
		refreshTTL, err := parseDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour)
		if err != nil {
			log.Fatalf("Invalid JWT_REFRESH_TTL: %v", err)
		}

		authService, err = auth.NewService(authStore, auth.ServiceConfig{
			JWTSecret:  jwtSecret,
			JWTIssuer:  os.Getenv("JWT_ISSUER"),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
		})
		if err != nil {
			log.Fatalf("Failed to init auth service: %v", err)
		}
	} else {
		log.Println("Auth disabled: JWT_SECRET is empty")
	}

	cookieSecure := strings.EqualFold(strings.TrimSpace(os.Getenv("AUTH_COOKIE_SECURE")), "true")
	guestMaxPages, err := parsePositiveIntEnv("TASK_MAX_PAGES_GUEST", 20)
	if err != nil {
		log.Fatalf("Invalid TASK_MAX_PAGES_GUEST: %v", err)
	}
	userMaxPages, err := parsePositiveIntEnv("TASK_MAX_PAGES_USER", 40)
	if err != nil {
		log.Fatalf("Invalid TASK_MAX_PAGES_USER: %v", err)
	}
	hardMaxPages, err := parsePositiveIntEnv("TASK_MAX_PAGES_HARD", 100)
	if err != nil {
		log.Fatalf("Invalid TASK_MAX_PAGES_HARD: %v", err)
	}

	log.Printf("Task quota config: guest=%d user=%d hard=%d", guestMaxPages, userMaxPages, hardMaxPages)

	// 创建并启动 HTTP 服务
	server := api.NewServer(tm, authService, cookieSecure, api.TaskQuotaConfig{
		GuestMaxPages: guestMaxPages,
		UserMaxPages:  userMaxPages,
		HardMaxPages:  hardMaxPages,
	})
	log.Println("Server starting on :8080")
	if err := server.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return time.ParseDuration(raw)
}

func parsePositiveIntEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("must be > 0")
	}
	return value, nil
}
