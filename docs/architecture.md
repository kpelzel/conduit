# Architecture

This document describes the internal architecture of Conduit and the responsibilities of its major components.

## High-Level System Layout

![Conduit Architecture Overview](images/conduit-architecture-simple.svg)

Conduit uses a distributed architecture with three main component types:

1. **Conduit Server** - Central coordinator that manages transfer requests and orchestrates work
2. **Conduit Runner** - Worker node daemon that executes transfer jobs and reports status
3. **Conduit FTA (File Transfer Agent)** - Transfer execution process that performs actual file operations with user-level permissions

Supporting infrastructure includes:

- **etcd** - Distributed key-value store for coordination and shared state
- **rqlite** - Distributed SQLite for persistent storage of completed transfer records

## Detailed System Layout

![Conduit Server Internal Architecture](images/conduit-architecture-detailed.svg)

## Server Components

The Conduit server consists of several internal subsystems that work together to manage the transfer lifecycle:

### gRPC Server

The gRPC server provides the primary API for client interactions. It handles:

- User authentication via Kerberos (using keytabs)
- Mutual TLS (mTLS) for secure communication
- API endpoints for transfer operations (create, status, abort, pause, resume, etc.)
- User identity translation via LDAP when configured

The server maintains in-memory maps of active transfers.

### Scheduler

The scheduler determines which transfers to execute and allocates resources across the worker pool. Key responsibilities:

- Orders pending transfers based on priority and submission time
- Tracks available memory and job capacity on each runner node
- Selects appropriate runner nodes based on resource requirements and node capabilities
- Maintains real-time status of all runner nodes via gRPC streams
- Sends transfer jobs to selected runner nodes for execution

The scheduler continuously receives status updates from runner nodes about available memory and running jobs, using this information to make scheduling decisions.

### Transfer Worker

A Transfer worker progresses transfers through their lifecycle by reacting to state changes and submitting work to the scheduler. Each transfer worker:

- Watches etcd for transfer state changes
- Progresses transfers to the next stage when a phase completes (e.g., INIT_COMPLETE → VALIDATION_READY → VALIDATION_SUBMITTED)
- Submits scheduler jobs for each transfer phase (validation, setup, data transfer, teardown)
- Manages lease acquisition for conflicting path access
- Handles transfer errors by setting error states in etcd
- Handles transfer aborts by cleaning up and marking transfers as aborted

Multiple transfer workers can run concurrently (primarily for testing concurrency with multiple conduit servers), but a single worker is typically sufficient as the progression logic is lightweight.

### Lease Watchdog

The lease watchdog monitors transfer liveness and detects failures. It:

- Watches etcd leases associated with each active transfer
- Detects when runner nodes or FTA processes fail (lease expiration)
- Removes expired transfers from scheduler queues
- In some cases, rolls back transfer state to allow retry (e.g., if expired while waiting for lease, rolls back to validation complete)

The watchdog monitors expiry timestamps stored in etcd to implement distributed failure detection without requiring direct health checks.

### etcd Manager

The etcd manager provides an abstraction layer for all etcd operations. It handles:

- Client connection management with automatic reconnection
- Key-value operations (get, put, delete) with proper namespacing
- Lease creation and renewal for liveness tracking
- Watch operations for monitoring key changes
- Transaction support for atomic updates
- Certificate-based authentication for secure etcd communication

All server components access etcd through this manager to ensure consistent error handling and connection pooling.

### rqlite Manager

The rqlite manager persists finalized transfer records to a distributed SQLite database. It:

- Stores completed transfer metadata for historical queries
- Provides SQL query interface for transfer history

Transfer records are only written to rqlite after successful completion or terminal failure, ensuring the system of record for active transfers remains in etcd.

### HTTP Server

Currently disabled. Reserved for potential future web interface.

## Data Transfer Execution

### Runner

The conduit-runner daemon runs on each worker node and serves as the execution agent. It:

- Registers with the conduit server cluster and maintains a persistent gRPC stream
- Reports node status (available memory, running job count) to schedulers
- Receives transfer job assignments from schedulers
- Spawns FTA processes with appropriate user credentials to execute transfers
- Monitors FTA process lifecycle and collects exit status

### FTA (File Transfer Agent)

The conduit-fta process performs the actual file transfer operations. Each FTA:

- Runs with the submitting user's UID/GID to enforce filesystem permissions
- Receives transfer parameters (source, destination, options) from the runner
- Loads appropriate transfer plugin (e.g., pftool, marchive) based on configuration
- Executes the transfer using the selected plugin
- Streams progress updates to etcd for monitoring
- Maintains an etcd lease to indicate liveness during execution
- Reports completion status and statistics back through etcd

FTA processes are short-lived, spawned for each transfer job and terminated upon completion. They receive necessary certificates via stdin from the runner to authenticate with etcd securely.

### Transfer Plugins

Conduit uses a plugin architecture for transfer execution, allowing different tools to be used based on requirements:

- **pftool** - Parallel file transfer tool optimized for HPC environments
- **marchive** - Archive-based transfer mechanism
- **rsync** - Standard file synchronization tool
- Custom plugins can be added by implementing the plugin interface

Plugins are loaded dynamically by the FTA process based on transfer configuration.

## Scalability and High Availability

### Server Clustering

Multiple conduit-server instances can run concurrently, each operating independently:

- Schedulers coordinate via etcd to avoid double-scheduling
- Transfer workers partition the transfer space to minimize overlap
- Watchdogs coordinate lease monitoring to prevent duplicate cleanup
- Servers can be added or removed dynamically
### etcd Cluster

etcd provides distributed coordination with:

- Raft consensus for strong consistency
- Automatic leader election
- Fault tolerance (tolerates (n-1)/2 failures in n-node cluster)

### rqlite Cluster

rqlite provides distributed persistence with:

- Raft consensus for SQLite replication
- Strong consistency for writes
- Eventually consistent reads from followers