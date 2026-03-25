package task

import (
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	//go:embed sql/claim_queued.sql
	claimQueuedQuery string
	//go:embed sql/mark_running.sql
	markRunningQuery string
	//go:embed sql/mark_failed.sql
	markFailedQuery string
	//go:embed sql/mark_completed.sql
	markCompletedQuery string
	//go:embed sql/insert_status.sql
	insertStatusQuery string
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}
