CREATE TYPE task_state AS ENUM (
    'QUEUED',
    'ASSIGNED',
    'RUNNING',
    'FAILED',
    'WORKER_LOST',
    'COMPLETED'
);

CREATE TABLE task (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    executable  TEXT        NOT NULL, -- S3 URI of the executable
    input_uri   TEXT,                 -- S3 URI of the input
    output_uri  TEXT,                 -- S3 URI where output will be written
    args        TEXT[]      NOT NULL DEFAULT '{}',
    max_retries INT         NOT NULL DEFAULT 3,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE task_status (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    task_id     UUID        NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    status      task_state  NOT NULL,
    assigned_to TEXT,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_status_task_created
    ON task_status (task_id, created_at DESC);

CREATE INDEX idx_task_status_assigned
    ON task_status (assigned_to)
    WHERE assigned_to IS NOT NULL;

CREATE INDEX idx_task_status_failed
    ON task_status (task_id)
    WHERE status = 'FAILED';
