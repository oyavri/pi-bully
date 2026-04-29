SELECT EXISTS (
    SELECT 1
    FROM task_lease
    WHERE task_id = $1 AND worker_id = $2
);
