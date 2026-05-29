## Context

The original MVP design for this repository chose PostgreSQL for persistent mirror configuration storage and for a `SKIP LOCKED`-based sync job queue, and the repository already contains SQL migrations plus a `docker-compose` database service. The current runtime does not follow that design: `cmd/mirror/main.go` constructs in-memory stores for users, mirror configurations, and sync jobs in both the server and worker entrypoints. As a result, restart durability is absent, the standalone worker process cannot share state with the server process, and local debugging can be misleading because a running Postgres container is not actually used.

This change needs a design because it crosses process boundaries, introduces a real runtime dependency on PostgreSQL, and affects startup, authentication bootstrapping, queue semantics, and deployment expectations.

## Goals / Non-Goals

**Goals:**
- Make mirror configurations and sync jobs durable across service restarts.
- Ensure the HTTP server, embedded worker, and standalone worker all share the same persistence layer through PostgreSQL.
- Reuse the existing store interfaces so higher-level handlers and worker orchestration remain largely unchanged.
- Apply the existing SQL migrations before normal runtime operations.
- Preserve in-memory stores for fast unit tests and low-friction isolated test setup.

**Non-Goals:**
- Replacing Basic Auth with a different authentication mechanism.
- Reworking the mirror sync algorithm, git cache structure, or webhook payload handling.
- Introducing Redis, external queue infrastructure, or background job orchestration outside PostgreSQL.
- Performing a one-time migration of previously in-memory mirror data; that data is already ephemeral.

## Decisions

### 1. Add PostgreSQL-backed store implementations for users, mirror configs, and sync jobs
- **Decision**: Implement SQL-backed versions of `auth.UserStore`, `store.MirrorConfigStore`, and `store.SyncJobStore`, while leaving the existing in-memory implementations in place for tests.
- **Why**: This gives runtime durability without forcing broad handler rewrites because the existing interfaces already match the required CRUD and queue operations.
- **Alternatives considered**:
  - Replace interfaces with direct SQL calls in handlers and worker bootstrap: simpler short term, but would spread persistence concerns across the codebase.
  - Use SQLite instead: easier local setup, but does not match the existing deployment plan and is a weaker fit for multi-process queue claiming.

### 2. Treat PostgreSQL as the default runtime persistence layer whenever `DATABASE_URL` is configured
- **Decision**: Server and worker startup should initialize PostgreSQL stores from `DATABASE_URL`, and fail fast if the database cannot be reached in normal runtime mode.
- **Why**: Silent fallback to in-memory would recreate the current debugging confusion and make data durability unreliable.
- **Alternatives considered**:
  - Automatic fallback to in-memory on DB failure: improves resilience for toy usage, but dangerous for a service whose core value is durable configuration and queue state.
  - Keep in-memory as the default and add an optional Postgres mode: too easy to deploy incorrectly and lose data again.

### 3. Run SQL migrations during startup before serving or claiming jobs
- **Decision**: Add a startup migration step that applies the existing SQL files in order before the server starts listening or the worker starts polling.
- **Why**: The schema already exists as raw SQL files; wiring them into startup is the smallest coherent way to guarantee tables and indexes are present.
- **Alternatives considered**:
  - Require manual migration execution as a separate operator step: workable, but easy to miss in development and local deployment.
  - Introduce a full migration framework first: more tooling than needed for the current repository state.

### 4. Seed or reconcile the configured Basic Auth admin user in PostgreSQL at startup
- **Decision**: On startup, ensure a user record exists for `BASIC_AUTH_USERNAME`, update its password hash to match `BASIC_AUTH_PASSWORD`, and preserve admin privileges.
- **Why**: Current auth bootstrapping assumes an in-memory seeded admin. Without a DB-backed equivalent, the service would boot but not have a predictable login path.
- **Alternatives considered**:
  - Fail if the admin user does not already exist: operationally brittle for local and first-run usage.
  - Create the user once and never reconcile password changes: conflicts with the current environment-driven login model.

### 5. Implement sync job claiming with PostgreSQL row locking
- **Decision**: Back `ClaimNextJob` with a transaction that selects one eligible queued or retrying job using row-level locking and marks it running atomically.
- **Why**: This fulfills the repository’s original queue design intent and lets multiple workers coordinate safely through the same database.
- **Alternatives considered**:
  - Poll queued jobs without locking and rely on updates racing: unsafe under concurrency.
  - Keep a single embedded worker and avoid standalone workers: does not satisfy the intended deployment shape and limits operability.

## Risks / Trade-offs

- **[Database startup dependency]**: Service startup can now fail if PostgreSQL is unavailable. → **Mitigation**: Fail fast with clear logs and keep `docker-compose` / env wiring explicit.
- **[Migration drift]**: Raw SQL migrations may become harder to manage as schema evolves. → **Mitigation**: Apply them deterministically in order and keep this change scoped to runtime wiring, not migration-framework replacement.
- **[Credential reconciliation side effects]**: Resetting the admin password from environment variables can surprise operators if multiple instances use different values. → **Mitigation**: Document that all instances must share the same configured admin credentials in this MVP model.
- **[Higher integration test complexity]**: DB-backed runtime flows are slower and harder to test than in-memory stores. → **Mitigation**: Keep interface-level unit tests on in-memory stores and add focused integration coverage only where DB behavior matters.

## Migration Plan

1. Add PostgreSQL store implementations and startup wiring behind `DATABASE_URL`.
2. Apply SQL migrations on startup before serving HTTP or polling jobs.
3. Seed/reconcile the configured admin user in PostgreSQL during bootstrap.
4. Start server and worker against the same database in local `docker-compose` and verify create/edit/sync flows persist across restarts.
5. Rollback path: redeploy the previous binary if needed. Runtime-created Postgres data can remain unused by the rollback build because the older in-memory version ignores it.

## Open Questions

- Should startup always require `DATABASE_URL`, or should there be an explicit development-only flag to opt into in-memory mode?
- Do we want a dedicated `migrate` subcommand later, or is startup-applied SQL sufficient for the current MVP?
- Should the embedded worker inside `server` remain enabled once standalone worker mode becomes reliable, or should that become configurable?
