package task

import (
	"context"

	"github.com/google/uuid"
)

type Store interface {
	ClaimQueued(ctx context.Context, workerID string, batchSize int) ([]Task, error)
	MarkRunning(ctx context.Context, taskID uuid.UUID, workerID string) error
	MarkCompleted(ctx context.Context, taskID uuid.UUID, workerID string) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, workerID string) error
}
