# Design

## System Model

Every node runs the same application and participates in cluster membership, leader election, and scheduling or execution depending on the current role.

The system has three main shared subsystems:
- PostgreSQL for task metadata and state
- object storage for scripts and artifacts
- cluster membership for liveness and peer discovery

## Roles

Any node may become leader. When a node becomes leader, it acts as the scheduler. The leader does not receive new task assignments while it remains leader. Non-leader nodes may receive and execute tasks.

If leadership changes, the new leader takes over scheduling responsibilities and stale work is recovered through the recovery logic rather than relying on old in-memory execution state.

### Scheduler

The scheduler is responsible for recovering stale scheduling state, recovering expired leases, recovering tasks during leader takeover, claiming queued tasks from PostgreSQL, and assigning tasks to free workers.

The scheduler enforces one active task per worker by checking active leases before assigning more work.

### Worker

A worker is responsible for accepting task assignments, transitioning tasks to `RUNNING`, renewing leases periodically, executing the task through the task executor, reporting success or failure. 

Workers do not write task state directly to PostgreSQL, instead, each task state transitions such as `RUNNING`, `COMPLETED`, and `FAILED` are reported to the scheduler. This approach keeps task state coordination centralized.

## Task Executor

Tasks are currently standardized as Python scripts stored in object storage.

Execution flow:
1. download the Python script (task)
2. download the input data using `input_uri` if it is provided
3. run the tasks using `python3 <script_name> ...args`
4. provide the path of input and output via environment variables passed as `TASK_INPUT` and `TASK_OUTPUT` to the script
5. if `output_uri` is provided, upload the output of the task

This keeps the scheduler and worker runtime decoupled from the internal behavior of the task itself.

## Storage Model

The system does not require a rigid bucket layout, it only depends on URI-based access to scripts, inputs, and outputs. 

The current project uses a simple convention for demonstration purposes.

Example URIs: 
```text
s3://bully/tasks/transcode.py
s3://bully/input/sample.mp4
s3://bully/output/sample-360p.mp4
```

## Database Model

PostgreSQL stores both task metadata and operational state.

The important split is:
- `task_status`: append-only state history
- `task_lease`: current active ownership/lease state

Because task state is stored as append-only history, the system can distinguish execution failure from worker loss and calculate retry usage without overwriting previous state.

## Task Lifecycle
Tasks follow different state flows depending on whether the execution succeeds, fails, or it is interrupted by worker loss. 

A successful execution follows this path:
```text
QUEUED -> SCHEDULING -> ASSIGNED -> RUNNING -> COMPLETED
```

A task may fail due to an execution error, such as an invalid script or runtime failure. If retry budget remains, it is requeued:
```text
QUEUED -> SCHEDULING -> ASSIGNED -> RUNNING -> FAILED -> QUEUED
```

Worker-related failure does not consume retry budget. If a worker disappears while holding a leased task, the task is marked as WORKER_LOST and requeued:
```text
QUEUED -> SCHEDULING -> ASSIGNED -> RUNNING -> WORKER_LOST -> QUEUED
```

In summary:
- `FAILED` tasks consumes retry budget
- `WORKER_LOST` does not consume retry budget. 
Once the retry limit is exhausted, the latest state remains `FAILED`.

## Lease And Recovery Model

Assigned or running tasks are protected by leases. While work is active workers renew the lease periodically through the scheduler. If a worker fails, renewal for the lease stops and at some point it expires. The task is marked as `WORKER_LOST` and is requeued by the scheduler.

There are two important recovery modes, lease expiry recovery and leader takeover recovery. 

In lease expiry recovery, the leader remains alive but a worker dies or stops renewing. The resulting action in the cluster is having the task is recovered after lease expiry.

In leader takeover recovery, current leader dies and a new leader takes over. Then, as a result, new leader reconciles the active work and abandoned or invalid in-flight tasks are requeued.

## Task Contract

Each task includes the following fields:
- `executable`
- `input_uri`
- `output_uri`
- `args`
- `max_retries`

For the tasks, the executor passes two environment variable for use: `TASK_INPUT` and `TASK_OUTPUT`. Both are optional, they may contain a local path to the input/output respectively or an empty string. 

If `output_uri` is set by the task, the script is expected to write the output into the path at `TASK_OUTPUT` and if it is not provided, the task is treated as side effect only and no upload is done.

## Limitations

- The system assumes the network is reliable and network partition is out of scope.
- Current execution model is centered on Python scripts. 
- The system assumes one cluster per deployment/database namespace.
- Cluster membership and database state are not globally atomic.
- There is no user-facing task submission API.
