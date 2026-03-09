package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	store "github.com/neyuki778/LLM-PDF-OCR/internal/store"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

const (
	taskKeyPrefix      = "task:"
	userTasksKeyPrefix = "user_tasks:"
	defaultHistorySize = 20
)

func (s *RedisStore) GetTask(ctx context.Context, id string) (*store.TaskRecord, error) {
	if id == "" {
		return nil, fmt.Errorf("ID should not be empty")
	}

	key := taskKeyPrefix + id
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
	key := taskKeyPrefix + task.ID
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
	key := taskKeyPrefix + id
	return s.client.Del(ctx, key).Err()
}

// AddUserTaskHistory adds a task id into a user's history index.
// The index is a ZSET sorted by creation time (Unix timestamp).
func (s *RedisStore) AddUserTaskHistory(ctx context.Context, userID, taskID string, createdAt time.Time) error {
	if userID == "" {
		return fmt.Errorf("userID should not be empty")
	}
	if taskID == "" {
		return fmt.Errorf("taskID should not be empty")
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	key := userTasksKeyPrefix + userID
	member := redis.Z{
		Score:  float64(createdAt.Unix()),
		Member: taskID,
	}
	return s.client.ZAdd(ctx, key, member).Err()
}

// ListUserTaskHistory returns user's task ids in reverse chronological order.
// When cursor is zero-value, it starts from the latest record.
func (s *RedisStore) ListUserTaskHistory(
	ctx context.Context,
	userID string,
	cursor time.Time,
	limit int64,
) ([]store.UserTaskHistoryEntry, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID should not be empty")
	}
	if limit <= 0 {
		limit = defaultHistorySize
	}

	maxScore := "+inf"
	if !cursor.IsZero() {
		maxScore = strconv.FormatInt(cursor.Unix(), 10)
	}

	key := userTasksKeyPrefix + userID
	opt := &redis.ZRangeBy{
		Min:    "-inf",
		Max:    maxScore,
		Offset: 0,
		Count:  limit,
	}

	rows, err := s.client.ZRevRangeByScoreWithScores(ctx, key, opt).Result()
	if err != nil {
		return nil, err
	}

	entries := make([]store.UserTaskHistoryEntry, 0, len(rows))
	for _, row := range rows {
		taskID, ok := row.Member.(string)
		if !ok || taskID == "" {
			continue
		}
		entries = append(entries, store.UserTaskHistoryEntry{
			TaskID:    taskID,
			CreatedAt: time.Unix(int64(row.Score), 0).UTC(),
		})
	}
	return entries, nil
}

func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}
