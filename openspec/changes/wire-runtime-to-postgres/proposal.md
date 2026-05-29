## Why

The service is currently running entirely on in-memory stores, so mirror configurations and sync jobs disappear on process restart and separate server/worker processes cannot reliably share state. This blocks the intended deployment model in `docker-compose`, causes misleading debugging around "database" state, and undermines the durability expected from a mirror service.

## What Changes

- Replace runtime use of in-memory mirror and sync job stores with PostgreSQL-backed implementations wired from `DATABASE_URL`.
- Persist the local admin user in PostgreSQL so authentication survives restarts and matches the configured basic auth credentials.
- Run the existing SQL migrations as part of bringing up the service so the database schema exists before handling requests or jobs.
- Move job claiming for worker processes to PostgreSQL-backed queue semantics so standalone workers and the embedded server worker see the same jobs.
- Keep the current store interfaces and in-memory implementations for tests and lightweight local unit coverage.

## Capabilities

### New Capabilities
- `runtime-persistence`: Durable runtime storage and shared job queue behavior for mirror server and worker processes backed by PostgreSQL.

### Modified Capabilities
- `mirror-config`: Mirror configurations and related sync activity must persist across restarts and remain accessible to any server or worker process connected to the same database.

## Impact

- Affected code: `cmd/mirror`, `internal/store`, `internal/auth`, worker job-claim/update paths, startup/bootstrap flow, and deployment wiring.
- Affected systems: PostgreSQL becomes a required runtime dependency for normal server/worker operation.
- Operational impact: local and deployed environments must provide `DATABASE_URL`, run migrations, and keep server/worker pointed at the same database.
