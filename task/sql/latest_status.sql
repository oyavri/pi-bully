SELECT status
FROM task_status
WHERE task_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;
