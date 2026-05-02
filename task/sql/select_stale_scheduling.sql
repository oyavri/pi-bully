WITH latest AS (
    SELECT DISTINCT ON (ts.task_id)
        ts.task_id,
        ts.status
    FROM task_status ts
    ORDER BY ts.task_id, ts.created_at DESC, ts.id DESC
)
SELECT t.id
FROM task t
JOIN latest l ON l.task_id = t.id
LEFT JOIN task_lease tl ON tl.task_id = t.id
WHERE l.status = 'SCHEDULING'
  AND tl.task_id IS NULL
FOR UPDATE OF t SKIP LOCKED;
