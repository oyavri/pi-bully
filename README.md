# pi-bully

## Overview
pi-bully is a fault tolerant distributed task execution system designed for clusters of low-power and low-resource devices such as Raspberry Pis. 

This project was developed as a graduation project and it focuses on leader election, distributed task execution, and recovery from follower or leader failures. The system assumes that the cluster network is reliable. It is designed to recover from node failures. Network partitions and split brain scenarios are outside of the current project scope. 

The system elects a leader with the Bully algorithm. The elected leader acts as the scheduler. The scheduler assigns queued tasks to free (alive & ready for computation) workers. The system recovers the tasks when workers or leaders fail. Workers execute Python scripts, renew leases while the work is in progress and report the result (completion or failure) back to the scheduler.

## Architecture

At a high level, the system consists of:
- a cluster of nodes that all participate in election
- one elected leader acting as scheduler
- non-leader nodes acting as workers
- PostgreSQL for task metadata, task history, and active leases
- S3-compatible object storage for task scripts, inputs, and outputs

INSERT DIAGRAM HERE

## Task Representation

Tasks are represented in PostgreSQL and object storage.

The `task` table stores the task definition including:
- executable URI (in other words the Python script)
- input URI (optional)
- output URI (optional, reserved beforehand)
- task arguments
- retry limit

The `task_status` table stores append-only task state history and the `task_lease` table stores current active ownership for running work.

Task scripts, inputs, and outputs are stored in an S3-compatible storage.

## Requirements

- .env must be configured before startup
- Cluster nodes must have correct reachable seed/advertise addresses

For local development the following are required:
- Go >= 1.26.1 
- Docker along with Docker Compose
- Protocol Buffers compiler (`protoc`) if protobuf code needs to be regenerated

For task execution (already downloaded in the docker-compose.yml) the following are required in the node:
- Python 3
- `ffmpeg` for the example [transcoding task](./scripts/transcode.py)

## How to Run 
This section describes local Docker-based execution. 

1. Configure the `.env` file (an example configuration is in `.env.example`). Make sure database, storage, memberlist, scheduler and node specific addresses are correct.
2. Ensure cluster nodes can resolve or reach each other. Each node must know seed nodes and advertise reachable addresses so that the cluster can form correctly. 
3. Start the local environment (if not needed, nodes can be excluded from the compose file):
```sh
docker compose up --build -d
```
4. Upload task scripts and any required inputs to LocalStack (or any other S3-compatible storage that you use).
5. Insert queued tasks into PostgreSQL.
6. To check logs, you can run:
```sh
docker compose logs -f 
```
7. You can inspect the `task` table for the state of the tasks or check the outputs in LocalStack (or the object storage you use).

> Note: Before deploying to a real node, the same environment variables and node address configuration must be updated to reflect the actual cluster addresses.

## Improvements
- Task submission API/UI would make it easier to add tasks but for demo current state of the project is enough.
- Task execution can be sandboxed via containers (docker or LXC) as it currently allows a script to run anything on the device running this.

## Documentation

See:
- [docs/design.md](./docs/design.md)
- [docs/references.md](./docs/references.md)
