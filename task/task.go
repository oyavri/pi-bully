package task

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/oyavri/pi-bully/cluster"
)

type Lease struct {
	TaskID    uuid.UUID
	WorkerID  uint64
	ExpiresAt time.Time
}

type Store interface {
	ClaimQueued(ctx context.Context, schedulerID uint64, batchSize int) ([]Task, error)
	ActiveLeases(ctx context.Context) ([]Lease, error)
	RenewLease(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkAssigned(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkRunning(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkCompleted(ctx context.Context, taskID uuid.UUID, workerID uint64) error
	MarkFailed(ctx context.Context, taskID uuid.UUID, workerID uint64, errMsg string) error
	MarkWorkerLost(ctx context.Context, taskID uuid.UUID, workerID uint64, errMsg string) error
	RecoverExpiredLeases(ctx context.Context) (int, error)
	RecoverDeadWorkerLeases(ctx context.Context, alive map[uint64]cluster.Member) (int, error)
	RecoverStaleScheduling(ctx context.Context) (int, error)
}
