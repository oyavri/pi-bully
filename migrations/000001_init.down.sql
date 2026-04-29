-- Indexes
DROP INDEX IF EXISTS idx_task_status_failed;
DROP INDEX IF EXISTS idx_task_status_assigned;
DROP INDEX IF EXISTS idx_task_status_task_created;
DROP INDEX IF EXISTS idx_task_lease_worker_id;
DROP INDEX IF EXISTS idx_task_lease_expires_at;
-- Tables
DROP TABLE IF EXISTS task_lease;
DROP TABLE IF EXISTS task_status;
DROP TABLE IF EXISTS task;
-- Enums/Types
DROP TYPE IF EXISTS task_state;
