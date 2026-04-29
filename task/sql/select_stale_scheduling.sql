WITH latest AS (
    SELECT DISTINCT ON (task_id)
        task_id,
        status
    FROM task_status
    ORDER BY task_id, created_at DESC, id DESC
)
SELECT l.task_id
FROM latest l
LEFT JOIN task_lease tl ON tl.task_id = l.task_id
WHERE l.status = 'SCHEDULING'
  AND tl.task_id IS NULL
FOR UPDATE SKIP LOCKED;
