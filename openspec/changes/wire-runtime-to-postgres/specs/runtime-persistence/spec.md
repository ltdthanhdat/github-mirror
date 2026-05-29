## ADDED Requirements

### Requirement: Runtime state SHALL persist in PostgreSQL
The system SHALL store users, mirror configurations, and sync jobs in PostgreSQL for normal server and worker operation.

#### Scenario: Server creates durable runtime records
- **WHEN** an authenticated user creates a mirror configuration while the service is running with `DATABASE_URL`
- **THEN** the system stores the mirror configuration and initial sync job in PostgreSQL
- **AND** that data remains available to later requests after the server process restarts

### Requirement: Workers SHALL share a database-backed job queue
The system SHALL allow any server or worker process connected to the same PostgreSQL database to claim and update sync jobs from a shared queue.

#### Scenario: Standalone worker sees jobs created by server
- **WHEN** the HTTP server enqueues a sync job in PostgreSQL
- **THEN** a separate worker process connected to the same database can claim that job
- **AND** the job transitions to running without requiring in-memory state from the server process

### Requirement: Startup SHALL prepare the runtime schema before serving traffic
The system SHALL apply required SQL migrations before handling HTTP requests or claiming sync jobs.

#### Scenario: Service boots against an empty database
- **WHEN** the server or worker starts with a reachable but empty PostgreSQL database
- **THEN** the system applies the required schema migrations before entering normal operation
- **AND** mirror CRUD and sync job processing can proceed without manual table creation

### Requirement: Configured basic auth credentials SHALL map to a durable admin user
The system SHALL ensure the configured basic auth user exists in PostgreSQL with administrative access.

#### Scenario: Service starts with configured admin credentials
- **WHEN** the service starts with `BASIC_AUTH_USERNAME` and `BASIC_AUTH_PASSWORD`
- **THEN** the system creates or updates the corresponding PostgreSQL user record
- **AND** requests authenticated with those credentials continue to work after a process restart
