## Context

We are building a new GitHub mirror service from scratch. The existing prototype (flotwig/git-mirror-docker) is insufficient for production use due to lack of dynamic token management, UI, database persistence, and job queuing. The service must be secure, reliable, and operable by a small team.

## Goals / Non-Goals

**Goals:**
- Provide a self-hosted service to mirror GitHub repositories between accounts.
- Securely store and manage GitHub personal access tokens (or app tokens).
- Use webhooks to trigger near real-time syncs.
- Offer a simple UI for configuration and monitoring.
- Ensure idempotency and handle failures gracefully.
- Use mature, low-dependency technologies (Go, PostgreSQL, git CLI).

**Non-Goals:**
- Support for other Git providers (GitLab, Bitbucket) in MVP.
- Advanced repository filtering (path-based, file-based).
- Multi-region deployment or auto-scaling in MVP.
- Built-in metrics or tracing (can be added via sidecar).
- Support for mirroring wikis, releases, or issues (only git refs).

## Decisions

### 1. Language and Framework: Go with standard library
- **Why**: Go compiles to a single binary, has excellent stdlib for HTTP, and is easy to deploy. Avoids framework bloat.
- **Alternatives considered**: Node.js (too many dependencies), Rust (steeper learning curve), Python (single binary less trivial).
- **Trade-off**: Less rapid prototyping than Node.js, but better for long-term maintenance and deployment.

### 2. Database: PostgreSQL
- **Why**: Provides reliable storage, transactions, and the `SKIP LOCKED` feature for a robust job queue without needing Redis.
- **Alternatives considered**: SQLite (not concurrent enough), MySQL (similar but PostgreSQL chosen for familiarity), Redis + separate DB (adds complexity).
- **Trade-off**: Slightly more complex setup than SQLite, but necessary for concurrency and reliability.

### 3. Job Queue: PostgreSQL table with `SKIP LOCKED`
- **Why**: Avoids introducing a new technology (Redis) for MVP. Leverages existing DB for atomic job claiming.
- **Alternatives considered**: Redis/BEANSTALK, AWS SQS (if on cloud), RabbitMQ.
- **Trade-off**: May not scale to extremely high throughput as Redis, but sufficient for expected load (10s of jobs/sec).

### 4. Git Operations: Bare repository cache per mirror
- **Why**: Fresh clone per job is safe but slow for large repos. Bare mirror cache allows incremental fetches, reducing bandwidth and time.
- **Alternatives considered**: Fresh clone with cleanup (simple but slow), shared cache with complex locking.
- **Trade-off**: Requires file-level locking (`flock`) and periodic `git gc`, but greatly improves performance for active mirrors.

### 5. Token Encryption: AES-GCM with a static key
- **Why**: Simplicity for MVP. The key is stored in an environment variable (`APP_ENCRYPTION_KEY`).
- **Alternatives considered**: HashiCorp Vault, AWS KMS, per-user keys derived from password.
- **Trade-off**: If the key is leaked, all tokens are compromised. For MVP, acceptable; later can integrate with a KMS.

### 6. UI: HTMX with server-rendered HTML
- **Why**: Enables dynamic UI without writing JavaScript. Keeps the stack simple (Go templates + HTMX).
- **Alternatives considered**: React/Vue SPA (requires separate build and API), traditional server-side rendering (full reloads).
- **Trade-off**: Less rich than SPA, but sufficient for configuration and monitoring UI and much faster to develop.

### 7. Authentication: Basic Auth over HTTPS
- **Why**: Extremely simple to implement and sufficient for a self-hosted service behind a reverse proxy (which can provide TLS).
- **Alternatives considered**: OAuth2, JWT sessions, GitHub OAuth.
- **Trade-off**: Less user-friendly (browser popup), but avoids implementing session management. For internal/tools use, acceptable.

### 8. Webhook Security: Verify `X-Hub-Signature-256`
- **Why**: Ensures the request genuinely came from GitHub and was not tampered with.
- **Alternatives considered**: IP allowlisting (not reliable due to GitHub's changing IPs), no verification.
- **Trade-off**: Requires storing the webhook secret encrypted, but is a GitHub-recommended practice.

### 9. Concurrency: Per-mirror lock via `flock`
- **Why**: Prevents two workers from syncing the same mirror simultaneously, which could cause corruption or wasted effort.
- **Alternatives considered**: Database lock (row-level), distributed lock (Redis).
- **Trade-off**: Requires a shared filesystem for the lock file (provided by Docker volume), but is simple and effective.

### 10. Deployment: Docker Compose
- **Why**: Easy to develop and deploy locally and in production. Defines API, worker, and DB services.
- **Alternatives considered**: Kubernetes (overkill for MVP), bare metal/systemd.
- **Trade-off**: Requires Docker, but provides isolation and reproducibility.

## Risks / Trade-offs

- **[Token leakage via logs]**: If tokens are accidentally logged, they could be stolen. → **Mitigation**: Never log tokens; mask URLs in debug output; audit code for logging.
- **[Git cache corruption]**: If a worker crashes during a push, the bare repo could be left in an inconsistent state. → **Mitigation**: Use atomic reference updates where possible; allow recovery by re-cloning if needed (though unlikely with bare repo).
- **[Webhook signature verification failure]**: If the secret is rotated incorrectly, syncs will stop. → **Mitigation**: Allow secret rotation via UI; log verification failures clearly.
- **[Disk space usage]**: Bare caches for large repositories can consume significant disk. → **Mitigation**: Implement `git gc --auto` periodically; monitor disk usage; allow pruning of inactive mirrors.
- **[Basic Auth usability]**: Users may find browser login popups inconvenient. → **Mitigation**: Document that this is intended to be used behind a reverse proxy with proper authentication (e.g., OAuth2 proxy) in production.
- **[PostgreSQL queue under high load]**: If job volume is very high, the queue table could become a bottleneck. → **Mitigation**: Monitor queue latency; consider migrating to Redis-backed queue if needed (designed to be replaceable).

## Open Questions

- Should we support mirroring multiple branches/tags with different patterns per mirror? (Current design: single `branch_pattern` glob)
- How should we handle initial sync for very large repositories? (Current design: fetch all refs; could be slow)
- Do we need to support GitHub Apps as an alternative to personal tokens for better security and granularity?