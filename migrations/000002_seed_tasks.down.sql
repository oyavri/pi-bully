DELETE FROM task_status
WHERE task_id IN (
    SELECT id
    FROM task
    WHERE executable IN (
        's3://pi-bully/jobs/example-job-1.sh',
        's3://pi-bully/jobs/example-job-2.sh',
        's3://pi-bully/jobs/example-job-3.sh',
        's3://pi-bully/jobs/example-job-4.sh',
        's3://pi-bully/jobs/example-job-5.sh',
        's3://pi-bully/jobs/example-job-6.sh',
        's3://pi-bully/jobs/example-job-7.sh',
        's3://pi-bully/jobs/example-job-8.sh',
        's3://pi-bully/jobs/example-job-9.sh',
        's3://pi-bully/jobs/example-job-10.sh'
    )
);

DELETE FROM task
WHERE executable IN (
    's3://pi-bully/jobs/example-job-1.sh',
    's3://pi-bully/jobs/example-job-2.sh',
    's3://pi-bully/jobs/example-job-3.sh',
    's3://pi-bully/jobs/example-job-4.sh',
    's3://pi-bully/jobs/example-job-5.sh',
    's3://pi-bully/jobs/example-job-6.sh',
    's3://pi-bully/jobs/example-job-7.sh',
    's3://pi-bully/jobs/example-job-8.sh',
    's3://pi-bully/jobs/example-job-9.sh',
    's3://pi-bully/jobs/example-job-10.sh'
);
