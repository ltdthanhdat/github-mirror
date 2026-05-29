## Why

The current mirror workflow depends on manual syncs or a future public webhook endpoint, but the application does not yet offer a built-in way to keep mirrors up to date automatically when inbound webhooks are unavailable. Users need a temporary but durable per-repository scheduling option so each mirror can continue syncing without external webhook delivery.

## What Changes

- Add a per-mirror cron schedule field that users can set while creating or editing a mirror configuration.
- Add runtime scheduling that evaluates enabled mirrors with configured schedules and enqueues the same full-sync jobs used by manual sync.
- Validate and persist schedule values so invalid cron expressions are rejected before save.
- Show each mirror's configured schedule in the authenticated UI so users can confirm whether automatic sync is enabled.
- Prevent the scheduler from enqueuing redundant sync work when a mirror already has an active queued or running job.

## Capabilities

### New Capabilities
- `scheduled-mirror-sync`: Automatic per-mirror sync scheduling that evaluates stored cron expressions and enqueues full-sync jobs when mirrors become due.

### Modified Capabilities
- `mirror-config`: Mirror configuration create, edit, and detail flows include a persisted per-mirror cron schedule and validate schedule input.

## Impact

- Affected code: `cmd/mirror`, `internal/http`, `internal/ui`, `internal/store`, `internal/models`, and SQL migrations for `mirror_configs`.
- Affected runtime behavior: adds a scheduler process or runtime loop that queries stored mirror schedules and writes to the existing sync job queue.
- Affected tests: model/store coverage for schedule persistence, HTTP/UI validation coverage for create/edit flows, and runtime tests for due-job enqueue behavior.
