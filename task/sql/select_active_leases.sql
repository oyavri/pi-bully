SELECT task_id, worker_id
FROM task_lease
FOR UPDATE SKIP LOCKED;
