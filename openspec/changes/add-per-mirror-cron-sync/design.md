## Context

The current application stores mirror configuration and sync job state in PostgreSQL and already has a durable worker that claims sync jobs from the shared queue. Manual sync works by enqueueing a full-sync job, while the GitHub webhook endpoint is present but does not yet drive job creation or serve real deployments without a public endpoint. Adding per-mirror cron scheduling therefore fits best as another producer for the existing sync job queue rather than as a new sync path.

This change crosses UI, persistence, runtime process management, and queue behavior. It also introduces user-provided schedule expressions, which require clear validation and predictable execution semantics before implementation begins.

## Goals / Non-Goals

**Goals:**
- Let each mirror store its own optional automatic sync schedule.
- Reuse the existing sync job queue and worker flow instead of introducing a second execution path.
- Ensure invalid schedules are rejected during create and edit flows.
- Avoid duplicate queued work when a mirror is already being processed or waiting to be processed.
- Keep the feature usable before webhook delivery is available in real deployments.

**Non-Goals:**
- Implementing full webhook-triggered sync behavior.
- Supporting per-user or per-mirror timezones in the first version.
- Supporting advanced cron variants such as seconds fields or `@hourly` aliases.
- Reworking the sync algorithm, branch filtering semantics, or job retry policy.

## Decisions

### Add mirror-level schedule fields to persisted configuration
- **Decision:** Extend `mirror_configs` with an optional cron expression field and a scheduler bookkeeping timestamp that records the last scheduled run time.
- **Why:** The cron expression must survive restarts and be editable through the existing mirror lifecycle. A stored scheduler timestamp gives the runtime a durable way to avoid repeated enqueue attempts for the same due window across polling iterations or process restarts.
- **Alternatives considered:**
  - Derive due state only from `last_synced_at`: rejected because failed or skipped runs would distort schedule evaluation and make deduplication ambiguous.
  - Store schedules outside the mirror record: rejected because it fragments ownership and complicates CRUD flows for a feature that is part of mirror behavior.

### Introduce a dedicated scheduler runtime command
- **Decision:** Add a `scheduler` command alongside the existing `server` and `worker` commands.
- **Why:** The deployment already separates HTTP and worker concerns into distinct containers. A dedicated scheduler avoids duplicate scheduling when multiple HTTP servers are started and keeps responsibility boundaries clean.
- **Alternatives considered:**
  - Run scheduling inside the server process: simpler locally, but unsafe once the HTTP service is scaled horizontally.
  - Run scheduling inside the worker: possible, but it mixes queue production and queue consumption responsibilities and makes future scaling less explicit.

### Support a narrow, explicit cron format
- **Decision:** Accept only standard 5-field cron expressions using a single runtime timezone.
- **Why:** This keeps input validation and runtime behavior predictable for the first release while still covering the user’s per-repo scheduling need.
- **Alternatives considered:**
  - Support seconds or nicknames like `@hourly`: rejected because they add parser and UX complexity without solving the current problem.
  - Support per-mirror timezone selection: rejected for the first release because it expands the data model and validation surface too early.

### Treat UTC as the schedule evaluation timezone
- **Decision:** Evaluate stored cron expressions in UTC and document that choice in the form helper text and detail page.
- **Why:** UTC avoids container-host drift and daylight-saving surprises. It is also the most portable default for infrastructure-managed workloads.
- **Alternatives considered:**
  - Use server local time: rejected because deployments may move across hosts or images with different timezone configuration.
  - Use `Asia/Ho_Chi_Minh` or another business timezone: rejected because the product currently has no user or organization locale model to justify a fixed regional assumption.

### Reuse full-sync job enqueue semantics with active-job deduplication
- **Decision:** When a mirror is due, enqueue the same full-sync job shape used by `Sync Now`, but skip enqueueing if that mirror already has a `queued` or `running` job.
- **Why:** This keeps scheduled sync behavior aligned with manual sync behavior and avoids inflating the queue with redundant work when previous runs have not drained yet.
- **Alternatives considered:**
  - Add a new job type for scheduled sync: rejected because the worker does not need different execution semantics.
  - Allow multiple scheduled jobs to accumulate: rejected because it wastes queue capacity and can trigger stale repeated syncs.

## Risks / Trade-offs

- [Scheduler misses one or more scheduled windows during downtime] -> Record only the latest scheduled run and enqueue at most one catch-up job on the next polling cycle rather than replaying every missed interval.
- [Cron parsing or validation behavior is unclear to users] -> Restrict accepted syntax, validate on save, and show a concrete example in the UI.
- [A scheduler crash or restart could enqueue the same mirror twice] -> Persist a scheduler bookkeeping timestamp and combine it with active-job checks before enqueueing.
- [UTC may surprise users expecting local time] -> State the timezone explicitly in the form and detail UI.

## Migration Plan

1. Add a SQL migration extending `mirror_configs` with nullable schedule and scheduler bookkeeping columns.
2. Deploy application code that understands the new columns before enabling the scheduler command in deployment manifests.
3. Add the `scheduler` service to `docker-compose` or production deployment so scheduled enqueueing starts only after the new runtime is available.
4. Roll back by stopping the scheduler service first, then reverting application code if needed; the added nullable columns can remain in place without breaking older data.

## Open Questions

- Should disabled mirrors keep their stored cron expression for later reuse, or should disabling a mirror also clear the schedule in the UI flow?
- Should the first version permit an immediate catch-up enqueue after downtime, or should it wait until the next future cron boundary only?
