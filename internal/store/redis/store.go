package redis

import (
	"context"
	"time"

	store "github.com/neyuki778/LLM-PDF-OCR/internal/store"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	tll		time.Duration
}

func (s *RedisStore) GetTask(ctx context.Context, id string) (*store.TaskRecord, error)
func (s *RedisStore) SaveTask(ctx context.Context, task *store.TaskRecord) error
func (s *RedisStore) UpdateTaskStatus(ctx context.Context, id string, status string) error
func (s *RedisStore) DeleteTask(ctx context.Context, id string) error