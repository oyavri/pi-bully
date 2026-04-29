SELECT task_id, worker_id
FROM task_lease
WHERE expires_at < NOW()
ORDER BY expires_at ASC, task_id ASC
FOR UPDATE SKIP LOCKED;
