package store

import "context"

type Store interface {
	GetTask(ctx context.Context, id string) (*TaskRecord, error)
	SaveTask(ctx context.Context, task *TaskRecord) error
	UpdateTaskStatus(ctx context.Context, id string, status string) error
	DeleteTask(ctx context.Context, id string) error
}
