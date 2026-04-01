package task

import (
	"context"
	_ "embed"
	"fmt"

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
)

type PostgresStore struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewPostgresStore(pool *pgxpool.Pool, logger *zap.Logger) *PostgresStore {
	return &PostgresStore{pool: pool, logger: logger}
}

func (s *PostgresStore) ClaimQueued(ctx context.Context, workerID string, batchSize int) ([]Task, error) {
	log := s.log("ClaimQueued").With(
		zap.String("workerID", workerID),
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
		if err := insertStatus(ctx, tx, t.ID, StateAssigned, workerID); err != nil {
			log.Error("failed to mark the task as assigned",
				zap.String("taskID", t.ID.String()),
				zap.Error(err),
			)
			return nil, fmt.Errorf("claim queued: mark assigned %s: %w", t.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return nil, fmt.Errorf("claim queued: commit: %w", err)
	}

	log.Info("successfully claimed tasks", zap.Int("count", len(tasks)))
	return tasks, nil
}

func (s *PostgresStore) MarkRunning(ctx context.Context, taskID uuid.UUID, workerID string) error {
	log := s.log("MarkRunning").With(
		zap.String("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark running: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := insertStatus(ctx, tx, taskID, StateRunning, workerID); err != nil {
		log.Error("failed to mark the task as running", zap.Error(err))
		return fmt.Errorf("mark running: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return fmt.Errorf("mark running: commit: %w", err)
	}

	log.Info("task marked as running")
	return nil
}

func (s *PostgresStore) MarkCompleted(ctx context.Context, taskID uuid.UUID, workerID string) error {
	log := s.log("MarkCompleted").With(
		zap.String("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark completed: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := insertStatus(ctx, tx, taskID, StateCompleted, workerID); err != nil {
		log.Error("failed to mark the task as completed", zap.Error(err))
		return fmt.Errorf("mark completed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error("failed to commit", zap.Error(err))
		return fmt.Errorf("mark completed: commit: %w", err)
	}

	log.Info("task marked as completed")
	return nil
}

func (s *PostgresStore) MarkFailed(ctx context.Context, taskID uuid.UUID, workerID string) error {
	log := s.log("MarkFailed").With(
		zap.String("workerID", workerID),
		zap.String("taskID", taskID.String()),
	)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		log.Error("failed to begin transaction", zap.Error(err))
		return fmt.Errorf("mark failed: begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := insertStatus(ctx, tx, taskID, StateFailed, workerID); err != nil {
		log.Error("failed to mark the task as failed", zap.Error(err))
		return fmt.Errorf("mark failed: %w", err)
	}

	var failedCount, maxRetries int
	err = tx.QueryRow(ctx, countFailedQuery, taskID).Scan(&failedCount, &maxRetries)
	if err != nil {
		log.Error("failed to scan retry count", zap.Error(err))
		return fmt.Errorf("mark failed: count retries: %w", err)
	}

	// Requeue if max retry count is not reached
	if failedCount < maxRetries {
		if err := insertStatus(ctx, tx, taskID, StateQueued, ""); err != nil {
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

func insertStatus(ctx context.Context, tx pgx.Tx, taskID uuid.UUID, state State, assignedTo string) error {
	var assigned any
	if assignedTo != "" {
		assigned = assignedTo
	}

	_, err := tx.Exec(ctx, insertStatusQuery, taskID, string(state), assigned)
	if err != nil {
		return fmt.Errorf("insert status (%s -> %s): %w", taskID, state, err)
	}

	return nil
}

func (s *PostgresStore) log(method string) *zap.Logger {
	return s.logger.With(zap.String("method", method))
}
