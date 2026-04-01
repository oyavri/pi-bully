WITH
latest AS (
    SELECT DISTINCT ON (task_id)
        task_id,
        status
    FROM task_status
    ORDER BY task_id, created_at DESC
),
failed_counts AS (
    SELECT task_id, COUNT(*) AS failed
    FROM task_status
    WHERE status = 'FAILED'
    GROUP BY task_id
)
SELECT t.id, t.executable, t.input_uri, t.output_uri, t.args, t.max_retries, t.created_at
FROM task t
JOIN latest l ON l.task_id = t.id
LEFT JOIN failed_counts fc ON fc.task_id = t.id
WHERE l.status IN ('QUEUED', 'WORKER_LOST')
    AND COALESCE(fc.failed, 0) < t.max_retries
ORDER BY t.created_at ASC
LIMIT $1
FOR UPDATE OF t SKIP LOCKED
