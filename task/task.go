package task

import (
	"context"

	"github.com/google/uuid"
)

type Store interface {
	ClaimQueued(ctx context.Context, schedulerID uint64, batchSize int) ([]Task, error)
	RenewLease(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkAssigned(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkRunning(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkCompleted(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	RecoverExpiredLeases(ctx context.Context) (int, error)
	RecoverDeadWorkerLeases(ctx context.Context, alive map[uint64]struct{}) (int, error)
	RecoverStaleScheduling(ctx context.Context) (int, error)
}
