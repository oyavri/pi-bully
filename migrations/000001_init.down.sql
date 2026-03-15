-- Indexes
DROP INDEX IF EXISTS idx_task_status_failed;
DROP INDEX IF EXISTS idx_task_status_assigned;
DROP INDEX IF EXISTS idx_task_status_task_created;
-- Tables
DROP TABLE IF EXISTS task_status;
DROP TABLE IF EXISTS task;
-- Enums/Types
DROP TYPE IF EXISTS task_state;
