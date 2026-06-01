## 1. Persist per-mirror schedule data

- [x] 1.1 Add a SQL migration extending `mirror_configs` with nullable cron schedule and scheduler bookkeeping columns.
- [x] 1.2 Update mirror configuration models and PostgreSQL store code to read, write, and persist the new schedule fields.
- [x] 1.3 Extend store interfaces with the mirror and job queries needed by the scheduler, including listing eligible mirrors and detecting active jobs per mirror.

## 2. Implement scheduled enqueue runtime

- [x] 2.1 Add cron parsing and validation utilities for the supported 5-field schedule format and UTC execution semantics.
- [x] 2.2 Add a dedicated `scheduler` runtime command that polls persisted mirror schedules and determines when mirrors are due.
- [x] 2.3 Reuse the existing full-sync job shape to enqueue scheduled work while skipping mirrors that already have queued or running jobs.
- [x] 2.4 Update local deployment wiring so the scheduler process can run alongside the existing server and worker services.

## 3. Expose schedule management in the mirror UI

- [x] 3.1 Add the cron schedule field and helper text to the mirror create and edit forms.
- [x] 3.2 Validate cron schedule input in create and update handlers, returning clear errors for invalid expressions.
- [x] 3.3 Show the configured schedule and execution timezone in the mirror detail UI.

## 4. Verify behavior

- [x] 4.1 Add store and migration tests covering schedule persistence and scheduler state fields.
- [x] 4.2 Add handler and UI tests covering valid schedule save, invalid schedule rejection, and schedule visibility in detail pages.
- [x] 4.3 Add runtime tests covering due-schedule enqueue behavior and duplicate-job prevention when active work already exists.
