INSERT INTO task_lease (task_id, worker_id, expires_at)
VALUES ($1, $2, NOW() + ($3 * INTERVAL '1 second'))
ON CONFLICT (task_id) DO UPDATE
SET worker_id = EXCLUDED.worker_id,
    expires_at = EXCLUDED.expires_at,
    updated_at = NOW();
