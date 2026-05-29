## Why

We need a robust GitHub mirror service to automatically synchronize repositories between source and target GitHub accounts. Current solutions (like flotwig/git-mirror-docker) are insufficient for dynamic token handling, UI, DB, and multi-repo management. We require a solution that supports encrypted token storage, webhook-triggered syncs, job queuing, and a simple UI for configuration and monitoring.

## What Changes

- New Go-based mirror service with API, worker, and UI components
- PostgreSQL for storing mirror configurations and job queue
- HTMX-based server-rendered UI for configuration and monitoring
- GitHub webhook integration for triggering syncs
- Token encryption at rest using AES-GCM
- Bare Git repository cache for efficient incremental syncs
- Basic auth for protecting the API and UI
- Job retry mechanism with exponential backoff
- Log retention and cleanup

## Capabilities

### New Capabilities
- `mirror-config`: CRUD operations for mirror configurations (source/target repos, tokens, branch patterns, etc.)
- `webhook-receiver`: Endpoint to receive GitHub push events and enqueue sync jobs
- `sync-worker`: Background worker that processes sync jobs using git CLI
- `mirror-ui`: HTMX-based UI to manage mirrors and view sync logs
- `token-manager`: Secure storage and retrieval of GitHub tokens (encryption/decryption)
- `job-queue`: PostgreSQL-based queue with SKIP LOCKED for job distribution
- `git-cache`: Bare repository cache per mirror for efficient incremental fetches and pushes

### Modified Capabilities
- (None - this is a new service)

## Impact

- New Go modules in cmd/mirror, internal/* (http, mirror, store, crypto, auth, ui)
- New database schema (users, mirror_configs, sync_jobs tables)
- New Dockerfile and docker-compose.yml for deployment
- No breaking changes to existing systems (this is a new service)