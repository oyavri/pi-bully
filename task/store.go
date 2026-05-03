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
	"github.com/oyavri/pi-bully/cluster"
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
		return nil, fmt.Errorf("claim queued: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, claimQueuedQuery, batchSize)
	if err != nil {
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
			return nil, fmt.Errorf("claim queued: scan: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("claim queued: rows: %w", err)
	}

	if len(tasks) == 0 {
		return []Task{}, nil
	}

	for _, t := range tasks {
		if err := insertStatus(ctx, tx, t.ID, StateScheduling, &schedulerID, ""); err != nil {
			return nil, fmt.Errorf("claim queued: mark scheduling %s: %w", t.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("claim queued: commit: %w", err)
	}

	log.Info("successfully claimed tasks", zap.Int("count", len(tasks)))
	return tasks, nil
}

func (s *PostgresStore) RenewLease(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	leaseDuration := int64(s.leaseDuration / time.Second)

	tag, err := s.pool.Exec(ctx, renewLeaseQuery, taskID, int64(workerID), leaseDuration)
	if err != nil {
		return fmt.Errorf("renew lease: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("renew lease: task %s is not leased to worker %d", taskID, workerID)
	}

	return nil
}

func (s *PostgresStore) MarkAssigned(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
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
		return fmt.Errorf("mark assigned: %w", err)
	}

	leaseDuration := int64(s.leaseDuration / time.Second)
	_, err = tx.Exec(ctx, upsertLeaseQuery, taskID, int64(workerID), leaseDuration)
	if err != nil {
		return fmt.Errorf("mark assigned: upsert lease: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("mark assigned: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkRunning(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mark running: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLeaseOwner(ctx, tx, taskID, workerID); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateAssigned, StateRunning); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateRunning, &workerID, ""); err != nil {
		return fmt.Errorf("mark running: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("mark running: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkCompleted(ctx context.Context, taskID uuid.UUID, workerID uint64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mark completed: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLeaseOwner(ctx, tx, taskID, workerID); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateRunning); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateCompleted, &workerID, ""); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}

	if _, err := tx.Exec(ctx, deleteLeaseQuery, taskID, int64(workerID)); err != nil {
		return fmt.Errorf("mark completed: delete lease: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("mark completed: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkFailed(ctx context.Context, taskID uuid.UUID, workerID uint64, errMsg string) error {
	log := s.log("MarkFailed").With(
		zap.Uint64("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mark failed: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLeaseOwner(ctx, tx, taskID, workerID); err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateAssigned, StateRunning); err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateFailed, &workerID, errMsg); err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}

	if _, err := tx.Exec(ctx, deleteLeaseQuery, taskID, int64(workerID)); err != nil {
		return fmt.Errorf("mark failed: delete lease: %w", err)
	}

	var failedCount, maxRetries int
	err = tx.QueryRow(ctx, countFailedQuery, taskID).Scan(&failedCount, &maxRetries)
	if err != nil {
		return fmt.Errorf("mark failed: count retries: %w", err)
	}

	// Requeue if max retry count is not reached
	if failedCount < maxRetries {
		if err := insertStatus(ctx, tx, taskID, StateQueued, nil, ""); err != nil {
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
		return fmt.Errorf("mark failed: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) MarkWorkerLost(ctx context.Context, taskID uuid.UUID, workerID uint64, errMsg string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mark worker lost: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureLatestStatusIn(ctx, tx, taskID, StateAssigned, StateRunning); err != nil {
		return fmt.Errorf("mark worker lost: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateWorkerLost, &workerID, errMsg); err != nil {
		return fmt.Errorf("mark worker lost: %w", err)
	}

	if _, err := tx.Exec(ctx, deleteLeaseQuery, taskID, int64(workerID)); err != nil {
		return fmt.Errorf("mark worker lost: delete lease: %w", err)
	}

	if err := insertStatus(ctx, tx, taskID, StateQueued, nil, ""); err != nil {
		return fmt.Errorf("mark worker lost: requeue: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("mark worker lost: commit: %w", err)
	}

	return nil
}

func (s *PostgresStore) RecoverExpiredLeases(ctx context.Context) (int, error) {
	log := s.log("RecoverExpiredLeases")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("recover expired leases: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, selectExpiredLeasesQuery)
	if err != nil {
		return 0, fmt.Errorf("recover expired leases: query: %w", err)
	}
	defer rows.Close()

	var leases []leaseRow

	for rows.Next() {
		var taskID uuid.UUID
		var workerID uint64
		if err := rows.Scan(&taskID, &workerID); err != nil {
			return 0, fmt.Errorf("recover expired leases: scan: %w", err)
		}

		leases = append(leases, leaseRow{TaskID: taskID, WorkerID: workerID})
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("recover expired leases: rows: %w", err)
	}

	if len(leases) == 0 {
		return 0, nil
	}

	for _, lease := range leases {
		if err := insertStatus(ctx, tx, lease.TaskID, StateWorkerLost, &lease.WorkerID, "lease expired"); err != nil {
			return 0, fmt.Errorf("recover expired leases: mark worker lost %s: %w", lease.TaskID, err)
		}

		if _, err := tx.Exec(ctx, deleteLeaseQuery, lease.TaskID, int64(lease.WorkerID)); err != nil {
			return 0, fmt.Errorf("recover expired leases: delete lease %s: %w", lease.TaskID, err)
		}

		if err := insertStatus(ctx, tx, lease.TaskID, StateQueued, nil, ""); err != nil {
			return 0, fmt.Errorf("recover expired leases: requeue %s: %w", lease.TaskID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("recover expired leases: commit: %w", err)
	}

	log.Info("recovered expired leases", zap.Int("count", len(leases)))
	return len(leases), nil
}

func (s *PostgresStore) RecoverDeadWorkerLeases(ctx context.Context, alive map[uint64]cluster.Member) (int, error) {
	log := s.log("RecoverDeadWorkerLeases")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("recover dead worker leases: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, selectActiveLeasesQuery)
	if err != nil {
		return 0, fmt.Errorf("recover dead worker leases: query: %w", err)
	}
	defer rows.Close()

	var deadLeases []leaseRow
	for rows.Next() {
		var row leaseRow
		var workerID int64
		if err := rows.Scan(&row.TaskID, &workerID); err != nil {
			return 0, fmt.Errorf("recover dead worker leases: scan: %w", err)
		}
		row.WorkerID = uint64(workerID)

		if _, ok := alive[row.WorkerID]; !ok {
			deadLeases = append(deadLeases, row)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("recover dead worker leases: rows: %w", err)
	}

	if len(deadLeases) == 0 {
		return 0, nil
	}

	for _, lease := range deadLeases {
		if err := insertStatus(ctx, tx, lease.TaskID, StateWorkerLost, &lease.WorkerID, "worker not alive during leader recovery"); err != nil {
			return 0, fmt.Errorf("recover dead worker leases: mark worker lost %s: %w", lease.TaskID, err)
		}

		if _, err := tx.Exec(ctx, deleteLeaseQuery, lease.TaskID, int64(lease.WorkerID)); err != nil {
			return 0, fmt.Errorf("recover dead worker leases: delete lease %s: %w", lease.TaskID, err)
		}

		if err := insertStatus(ctx, tx, lease.TaskID, StateQueued, nil, ""); err != nil {
			return 0, fmt.Errorf("recover dead worker leases: requeue %s: %w", lease.TaskID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("recover dead worker leases: commit: %w", err)
	}

	log.Info("recovered dead worker leases", zap.Int("count", len(deadLeases)))
	return len(deadLeases), nil
}

func (s *PostgresStore) RecoverStaleScheduling(ctx context.Context) (int, error) {
	log := s.log("RecoverStaleScheduling")

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("recover stale scheduling: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, selectStaleSchedulingQuery)
	if err != nil {
		return 0, fmt.Errorf("recover stale scheduling: query: %w", err)
	}
	defer rows.Close()

	var taskIDs []uuid.UUID
	for rows.Next() {
		var taskID uuid.UUID
		if err := rows.Scan(&taskID); err != nil {
			return 0, fmt.Errorf("recover stale scheduling: scan: %w", err)
		}
		taskIDs = append(taskIDs, taskID)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("recover stale scheduling: rows: %w", err)
	}

	if len(taskIDs) == 0 {
		return 0, nil
	}

	for _, taskID := range taskIDs {
		if err := insertStatus(ctx, tx, taskID, StateQueued, nil, "recovered stale scheduling"); err != nil {
			return 0, fmt.Errorf("recover stale scheduling: requeue %s: %w", taskID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
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
