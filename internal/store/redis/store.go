package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	store "github.com/neyuki778/LLM-PDF-OCR/internal/store"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func (s *RedisStore) GetTask(ctx context.Context, id string) (*store.TaskRecord, error) {
	if id == "" {
		return nil, fmt.Errorf("ID should not be empty")
	}

	key := "task:" + id
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	var rec store.TaskRecord
	if err := json.Unmarshal([]byte(val), &rec); err != nil {
		return nil, err
	}

	return &rec, nil
}
func (s *RedisStore) SaveTask(ctx context.Context, task *store.TaskRecord) error {
	if task == nil {
		return fmt.Errorf("task should not be nil")
	}
	key := "task:" + task.ID
	val, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, val, s.ttl).Err()
}
func (s *RedisStore) UpdateTaskStatus(ctx context.Context, id string, status string) error {
	if id == "" {
		return fmt.Errorf("ID should not be empty")
	} else if status == "" {
		return fmt.Errorf("Status should not be empty")
	}

	val, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	val.Status = status
	val.UpdatedAt = time.Now()

	return s.SaveTask(ctx, val)
}
func (s *RedisStore) DeleteTask(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("ID should not be empty")
	}
	key := "task:" + id
	return s.client.Del(ctx, key).Err()
}

func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}
