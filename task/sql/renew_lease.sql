UPDATE task_lease
SET expires_at = NOW() + ($3 * INTERVAL '1 second'),
    updated_at = NOW()
WHERE task_id = $1 AND worker_id = $2;
