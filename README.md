# pi-bully
pi-bully is a distributed, fault-tolerant task orchestration system designed for Raspberry Pi clusters. It leverages the Bully algorithm for leader election and integrates with an object storage for persistent data handling and PostgreSQL for state management.

## Overview
The system allows users to submit computational tasks (executables, arguments, and data) via a REST API. A cluster of Raspberry Pis automatically elects a scheduler (leader) using the Bully algorithm. The scheduler delegates tasks to workers, who execute the work and upload results back to the object storage.

## How to run the project
TO DO.
