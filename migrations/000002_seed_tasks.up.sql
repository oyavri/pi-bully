INSERT INTO task (executable, input_uri, output_uri, args, max_retries)
VALUES
    (
        's3://pi-bully/jobs/example-job-1.sh',
        's3://pi-bully/input/task-1.txt',
        's3://pi-bully/output/task-1.out',
        ARRAY['--mode=test', '--count=10'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-2.sh',
        's3://pi-bully/input/task-2.txt',
        's3://pi-bully/output/task-2.out',
        ARRAY['--mode=test', '--count=20'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-3.sh',
        's3://pi-bully/input/task-3.txt',
        's3://pi-bully/output/task-3.out',
        ARRAY['--mode=test', '--count=30'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-4.sh',
        's3://pi-bully/input/task-4.txt',
        's3://pi-bully/output/task-4.out',
        ARRAY['--mode=test', '--count=40'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-5.sh',
        's3://pi-bully/input/task-5.txt',
        's3://pi-bully/output/task-5.out',
        ARRAY['--mode=test', '--count=50'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-6.sh',
        's3://pi-bully/input/task-6.txt',
        's3://pi-bully/output/task-6.out',
        ARRAY['--mode=test', '--count=60'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-7.sh',
        's3://pi-bully/input/task-7.txt',
        's3://pi-bully/output/task-7.out',
        ARRAY['--mode=test', '--count=70'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-8.sh',
        's3://pi-bully/input/task-8.txt',
        's3://pi-bully/output/task-8.out',
        ARRAY['--mode=test', '--count=80'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-9.sh',
        's3://pi-bully/input/task-9.txt',
        's3://pi-bully/output/task-9.out',
        ARRAY['--mode=test', '--count=90'],
        3
    ),
    (
        's3://pi-bully/jobs/example-job-10.sh',
        's3://pi-bully/input/task-10.txt',
        's3://pi-bully/output/task-10.out',
        ARRAY['--mode=test', '--count=100'],
        3
    );

INSERT INTO task_status (task_id, status)
SELECT t.id, 'QUEUED'::task_state
FROM task t
WHERE t.executable IN (
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
