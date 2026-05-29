## 1. Database Bootstrap

- [x] 1.1 Add a PostgreSQL connection/bootstrap path that reads `DATABASE_URL` and initializes shared runtime dependencies for server and worker startup.
- [x] 1.2 Implement ordered execution of the existing SQL migration files before the server begins serving requests or the worker begins polling jobs.
- [x] 1.3 Add PostgreSQL-backed user bootstrapping so the configured basic auth admin user is created or reconciled in durable storage.

## 2. Persistent Store Implementations

- [x] 2.1 Implement a PostgreSQL-backed `MirrorConfigStore` that supports create, get, list-by-user, update, and delete against `mirror_configs`.
- [x] 2.2 Implement a PostgreSQL-backed `SyncJobStore` that supports create, get, list-by-mirror, update, and atomic claim-next-job behavior using row locking.
- [x] 2.3 Implement a PostgreSQL-backed `auth.UserStore` compatible with the existing basic auth flow.

## 3. Runtime Wiring

- [x] 3.1 Update `cmd/mirror` server startup to use PostgreSQL-backed stores in normal runtime mode instead of in-memory mirror/job/user stores.
- [x] 3.2 Update standalone worker startup and shared worker construction so all processes use the same PostgreSQL-backed mirror/job state.
- [x] 3.3 Preserve in-memory store implementations for unit tests and any explicitly isolated test setup that should not require PostgreSQL.

## 4. Verification

- [x] 4.1 Add tests for PostgreSQL-backed store behavior, including durable mirror CRUD and shared sync job claiming semantics.
- [x] 4.2 Add or update integration coverage to verify mirror configuration data survives process restart when the service reconnects to the same database.
- [x] 4.3 Validate the local runtime path with `docker-compose` so server and worker share PostgreSQL-backed state and mirror flows continue to work end-to-end.
