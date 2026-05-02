package task

import (
	"context"
	_ "embed"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	//go:embed sql/claim_queued.sql
	claimQueuedQuery string
	//go:embed sql/insert_status.sql
	insertStatusQuery string
	//go:embed sql/count_failed.sql
	countFailedQuery string
	//go:embed sql/upsert_lease.sql
	upsertLeaseQuery string
	//go:embed sql/renew_lease.sql
	renewLeaseQuery string
	//go:embed sql/delete_lease.sql
	deleteLeaseQuery string
	//go:embed sql/check_lease_owner.sql
	checkLeaseOwnerQuery string
	//go:embed sql/latest_status.sql
	latestStatusQuery string
	//go:embed sql/select_expired_leases.sql
	selectExpiredLeasesQuery string
	//go:embed sql/select_active_leases.sql
	selectActiveLeasesQuery string
	//go:embed sql/select_stale_scheduling.sql
	selectStaleSchedulingQuery string
)

type PostgresStore struct {
	pool          *pgxpool.Pool
	leaseDuration time.Duration
	logger        *zap.Logger
}

func NewPostgresStore(pool *pgxpool.Pool, leaseDuration time.Duration, logger *zap.Logger) *PostgresStore {
	return &PostgresStore{
		pool:          pool,
		leaseDuration: leaseDuration,
		logger:        logger,
	}
}

func (s *PostgresStore) ClaimQueued(ctx context.Context, schedulerID uint64, batchSize int) ([]Task, error) {
	log := s.log("ClaimQueued").With(
		zap.Uint64("schedulerID", schedulerID),
		zap.Int("batchSize", batchSize),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("claim queued: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, claimQueuedQuery, batchSize)
	if err != nil {
		log.Error("failed to execute claim queued query", zap.Error(err))
		return nil, fmt.Errorf("claim queued: query: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(
			&t.ID,
			&t.Executable,
			&t.InputURI,
			&t.OutputURI,
			&t.Args,
			&t.MaxRetries,
			&t.CreatedAt,
		); err != nil {
			log.Error("failed to scan claimed task", zap.Error(err))
			return nil, fmt.Errorf("claim queued: scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		log.Error("row iteration error", zap.Error(err))
		return nil, fmt.Errorf("claim queued: rows: %w", err)
	}

	if len(tasks) == 0 {
		log.Debug("no queued tasks found")
		if err := tx.Commit(ctx); err != nil {
			log.Error("failed to commit empty transaction", zap.Error(err))
			return nil, fmt.Errorf("claim queued: commit: %w", err)
		}
		return []Task{}, nil
	}

	for _, t := range tasks {
		if err := insertStatus(ctx, tx, t.ID, StateScheduling, &schedulerID, ""); err != nil {
			log.Error("failed to mark the task as scheduling",
				zap.String("taskID", t.ID.String()),
				zap.Error(err),
			)
			return nil, fmt.Errorf("claim queued: mark scheduling %s: %w", t.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return nil, fmt.Errorf("claim queued: commit: %w", err)
	}

	log.Info("successfully claimed tasks", zap.Int("count", len(tasks)))
	return tasks, nil
}

func (s *PostgresStore) RenewLease(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	log := s.log("RenewLease").With(
		zap.Uint64("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	leaseDuration := int64(s.leaseDuration / time.Second)

	tag, err := s.pool.Exec(ctx, renewLeaseQuery, taskID, int64(workerID), leaseDuration)
	if err != nil {
		log.Error("failed to renew task lease", zap.Error(err))
		return fmt.Errorf("renew lease: %w", err)
	}

	if tag.RowsAffected() == 0 {
		err := fmt.Errorf("task %s is not leased to worker %d", taskID, workerID)
		log.Warn("lease renewal rejected", zap.Error(err))
		return fmt.Errorf("renew lease: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkAssigned(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	log := s.log("MarkAssigned").With(
		zap.Uint64("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark assigned: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	status, err := latestStatus(ctx, tx, taskID)
	if err != nil {
		return fmt.Errorf("mark assigned: %w", err)
	}
	if status != StateScheduling {
		return fmt.Errorf("mark assigned: invalid latest status %s for task %s", status, taskID)
	}

	if err := insertStatus(ctx, tx, taskID, StateAssigned, &workerID, ""); err != nil {
		log.Error("failed to insert assigned status", zap.Error(err))
		return fmt.Errorf("mark assigned: %w", err)
	}

	leaseDuration := int64(s.leaseDuration / time.Second)
	_, err = tx.Exec(ctx, upsertLeaseQuery, taskID, int64(workerID), leaseDuration)
	if err != nil {
		log.Error("failed to upsert task lease", zap.Error(err))
		return fmt.Errorf("mark assigned: upsert lease: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return fmt.Errorf("mark assigned: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkRunning(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	log := s.log("MarkRunning").With(
		zap.Uint64("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark running: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLeaseOwner(ctx, tx, taskID, workerID); err != nil {
		log.Warn("lease ownership validation failed", zap.Error(err))
		return fmt.Errorf("mark running: %w", err)
	}

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateAssigned, StateRunning); err != nil {
		log.Warn("latest status validation failed", zap.Error(err))
		return fmt.Errorf("mark running: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateRunning, &workerID, ""); err != nil {
		log.Error("failed to mark the task as running", zap.Error(err))
		return fmt.Errorf("mark running: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return fmt.Errorf("mark running: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkCompleted(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	log := s.log("MarkCompleted").With(
		zap.Uint64("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark completed: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLeaseOwner(ctx, tx, taskID, workerID); err != nil {
		log.Warn("lease ownership validation failed", zap.Error(err))
		return fmt.Errorf("mark completed: %w", err)
	}

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateRunning); err != nil {
		log.Warn("latest status validation failed", zap.Error(err))
		return fmt.Errorf("mark completed: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateCompleted, &workerID, ""); err != nil {
		log.Error("failed to mark the task as completed", zap.Error(err))
		return fmt.Errorf("mark completed: %w", err)
	}

	if _, err := tx.Exec(ctx, deleteLeaseQuery, taskID, int64(workerID)); err != nil {
		log.Error("failed to delete task lease", zap.Error(err))
		return fmt.Errorf("mark completed: delete lease: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return fmt.Errorf("mark completed: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkFailed(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	log := s.log("MarkFailed").With(
		zap.Uint64("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark failed: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLeaseOwner(ctx, tx, taskID, workerID); err != nil {
		log.Warn("lease ownership validation failed", zap.Error(err))
		return fmt.Errorf("mark failed: %w", err)
	}

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateAssigned, StateRunning); err != nil {
		log.Warn("latest status validation failed", zap.Error(err))
		return fmt.Errorf("mark failed: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateFailed, &workerID, ""); err != nil {
		log.Error("failed to mark the task as failed", zap.Error(err))
		return fmt.Errorf("mark failed: %w", err)
	}

	if _, err := tx.Exec(ctx, deleteLeaseQuery, taskID, int64(workerID)); err != nil {
		log.Error("failed to delete task lease", zap.Error(err))
		return fmt.Errorf("mark failed: delete lease: %w", err)
	}

	var failedCount, maxRetries int
	err = tx.QueryRow(ctx, countFailedQuery, taskID).Scan(&failedCount, &maxRetries)
	if err != nil {
		log.Error("failed to scan retry count", zap.Error(err))
		return fmt.Errorf("mark failed: count retries: %w", err)
	}

	// Requeue if max retry count is not reached
	if failedCount < maxRetries {
		if err := insertStatus(ctx, tx, taskID, StateQueued, nil, ""); err != nil {
			log.Error("failed to requeue failed task", zap.Error(err))
			return fmt.Errorf("mark failed: requeue: %w", err)
		}

		log.Info("task failed and requeued",
			zap.Int("failedCount", failedCount),
			zap.Int("maxRetries", maxRetries),
		)
	} else {
		log.Warn("task permanently failed, max retries reached",
			zap.Int("failedCount", failedCount),
			zap.Int("maxRetries", maxRetries),
		)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return fmt.Errorf("mark failed: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) RecoverExpiredLeases(ctx context.Context) (int, error) {
	log := s.log("RecoverExpiredLeases")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return 0, fmt.Errorf("recover expired leases: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, selectExpiredLeasesQuery)
	if err != nil {
		log.Error("failed to query expired leases", zap.Error(err))
		return 0, fmt.Errorf("recover expired leases: query: %w", err)
	}
	defer rows.Close()

	var leases []leaseRow

	for rows.Next() {
		var taskID uuid.UUID
		var workerID uint64
		if err := rows.Scan(&taskID, &workerID); err != nil {
			log.Error("failed to scan expired lease", zap.Error(err))
			return 0, fmt.Errorf("recover expired leases: scan: %w", err)
		}

		leases = append(leases, leaseRow{TaskID: taskID, WorkerID: workerID})
	}

	if err := rows.Err(); err != nil {
		log.Error("expired lease iteration error", zap.Error(err))
		return 0, fmt.Errorf("recover expired leases: rows: %w", err)
	}

	if len(leases) == 0 {
		return 0, nil
	}

	for _, lease := range leases {
		if err := insertStatus(ctx, tx, lease.TaskID, StateWorkerLost, &lease.WorkerID, "lease expired"); err != nil {
			log.Error(
				"failed to insert WORKER_LOST",
				zap.String("taskID", lease.TaskID.String()),
				zap.Error(err),
			)
			return 0, fmt.Errorf("recover expired leases: mark worker lost %s: %w", lease.TaskID, err)
		}

		if _, err := tx.Exec(ctx, deleteLeaseQuery, lease.TaskID, int64(lease.WorkerID)); err != nil {
			log.Error("failed to delete expired lease", zap.String("taskID", lease.TaskID.String()), zap.Error(err))
			return 0, fmt.Errorf("recover expired leases: delete lease %s: %w", lease.TaskID, err)
		}

		if err := insertStatus(ctx, tx, lease.TaskID, StateQueued, nil, ""); err != nil {
			log.Error("failed to requeue task after expired lease", zap.String("taskID", lease.TaskID.String()), zap.Error(err))
			return 0, fmt.Errorf("recover expired leases: requeue %s: %w", lease.TaskID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit recovery transaction", zap.Error(err))
		return 0, fmt.Errorf("recover expired leases: commit: %w", err)
	}

	log.Info("recovered expired leases", zap.Int("count", len(leases)))
	return len(leases), nil
}

func (s *PostgresStore) RecoverDeadWorkerLeases(ctx context.Context, alive map[uint64]struct{}) (int, error) {
	log := s.log("RecoverDeadWorkerLeases")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return 0, fmt.Errorf("recover dead worker leases: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, selectActiveLeasesQuery)
	if err != nil {
		log.Error("failed to query active leases", zap.Error(err))
		return 0, fmt.Errorf("recover dead worker leases: query: %w", err)
	}
	defer rows.Close()

	var deadLeases []leaseRow
	for rows.Next() {
		var row leaseRow
		var workerID int64
		if err := rows.Scan(&row.TaskID, &workerID); err != nil {
			log.Error("failed to scan active lease", zap.Error(err))
			return 0, fmt.Errorf("recover dead worker leases: scan: %w", err)
		}
		row.WorkerID = uint64(workerID)

		if _, ok := alive[row.WorkerID]; !ok {
			deadLeases = append(deadLeases, row)
		}
	}
	if err := rows.Err(); err != nil {
		log.Error("active lease iteration error", zap.Error(err))
		return 0, fmt.Errorf("recover dead worker leases: rows: %w", err)
	}

	if len(deadLeases) == 0 {
		return 0, nil
	}

	for _, lease := range deadLeases {
		if err := insertStatus(ctx, tx, lease.TaskID, StateWorkerLost, &lease.WorkerID, "worker not alive during leader takeover"); err != nil {
			log.Error("failed to insert WORKER_LOST", zap.String("taskID", lease.TaskID.String()), zap.Error(err))
			return 0, fmt.Errorf("recover dead worker leases: mark worker lost %s: %w", lease.TaskID, err)
		}

		if _, err := tx.Exec(ctx, deleteLeaseQuery, lease.TaskID, int64(lease.WorkerID)); err != nil {
			log.Error("failed to delete dead worker lease", zap.String("taskID", lease.TaskID.String()), zap.Error(err))
			return 0, fmt.Errorf("recover dead worker leases: delete lease %s: %w", lease.TaskID, err)
		}

		if err := insertStatus(ctx, tx, lease.TaskID, StateQueued, nil, ""); err != nil {
			log.Error("failed to requeue task from dead worker", zap.String("taskID", lease.TaskID.String()), zap.Error(err))
			return 0, fmt.Errorf("recover dead worker leases: requeue %s: %w", lease.TaskID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit dead-worker recovery transaction", zap.Error(err))
		return 0, fmt.Errorf("recover dead worker leases: commit: %w", err)
	}

	log.Info("recovered dead worker leases", zap.Int("count", len(deadLeases)))
	return len(deadLeases), nil
}

func (s *PostgresStore) RecoverStaleScheduling(ctx context.Context) (int, error) {
	log := s.log("RecoverStaleScheduling")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return 0, fmt.Errorf("recover stale scheduling: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, selectStaleSchedulingQuery)
	if err != nil {
		log.Error("failed to query stale scheduling tasks", zap.Error(err))
		return 0, fmt.Errorf("recover stale scheduling: query: %w", err)
	}
	defer rows.Close()

	var taskIDs []uuid.UUID
	for rows.Next() {
		var taskID uuid.UUID
		if err := rows.Scan(&taskID); err != nil {
			log.Error("failed to scan stale scheduling task", zap.Error(err))
			return 0, fmt.Errorf("recover stale scheduling: scan: %w", err)
		}
		taskIDs = append(taskIDs, taskID)
	}

	if err := rows.Err(); err != nil {
		log.Error("stale scheduling iteration error", zap.Error(err))
		return 0, fmt.Errorf("recover stale scheduling: rows: %w", err)
	}

	if len(taskIDs) == 0 {
		return 0, nil
	}

	for _, taskID := range taskIDs {
		if err := insertStatus(ctx, tx, taskID, StateQueued, nil, "recovered stale scheduling"); err != nil {
			log.Error(
				"failed to requeue stale scheduling task",
				zap.String("taskID", taskID.String()),
				zap.Error(err),
			)
			return 0, fmt.Errorf("recover stale scheduling: requeue %s: %w", taskID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit stale scheduling recovery transaction", zap.Error(err))
		return 0, fmt.Errorf("recover stale scheduling: commit: %w", err)
	}

	log.Info("recovered stale scheduling tasks", zap.Int("count", len(taskIDs)))
	return len(taskIDs), nil
}

func (s *PostgresStore) log(method string) *zap.Logger {
	return s.logger.With(zap.String("method", method))
}

func insertStatus(ctx context.Context, tx pgx.Tx, taskID uuid.UUID, state State, assignedTo *uint64, errMsg string) error {
	var assigned any
	if assignedTo != nil {
		assigned = int64(*assignedTo)
	}

	var errVal any
	if errMsg != "" {
		errVal = errMsg
	}

	_, err := tx.Exec(ctx, insertStatusQuery, taskID, string(state), assigned, errVal)
	if err != nil {
		return fmt.Errorf("insert status (%s -> %s): %w", taskID, state, err)
	}

	return nil
}

func latestStatus(ctx context.Context, tx pgx.Tx, taskID uuid.UUID) (State, error) {
	var status string
	err := tx.QueryRow(ctx, latestStatusQuery, taskID).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("latest status: %w", err)
	}

	return State(status), nil
}

func ensureLeaseOwner(ctx context.Context, tx pgx.Tx, taskID uuid.UUID, workerID uint64) error {
	var ok bool
	err := tx.QueryRow(ctx, checkLeaseOwnerQuery, taskID, int64(workerID)).Scan(&ok)
	if err != nil {
		return fmt.Errorf("check lease owner: %w", err)
	}

	if !ok {
		return fmt.Errorf("task %s is not leased to worker %d", taskID, workerID)
	}

	return nil
}

func ensureLatestStatusIn(ctx context.Context, tx pgx.Tx, taskID uuid.UUID, allowed ...State) error {
	status, err := latestStatus(ctx, tx, taskID)
	if err != nil {
		return err
	}

	if !slices.Contains(allowed, status) {
		return fmt.Errorf("task %s has invalid latest status %s", taskID, status)
	}

	return nil
}

type leaseRow struct {
	TaskID   uuid.UUID
	WorkerID uint64
}
