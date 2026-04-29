SELECT COUNT(*) FILTER (WHERE ts.status = 'FAILED') AS failed_count, t.max_retries
FROM task t
LEFT JOIN task_status ts ON ts.task_id = t.id
WHERE t.id = $1
GROUP BY t.max_retries;
