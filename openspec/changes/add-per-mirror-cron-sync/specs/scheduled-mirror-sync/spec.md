## ADDED Requirements

### Requirement: Mirrors can run on a persisted per-mirror schedule
The system SHALL evaluate enabled mirror configurations that have a stored cron schedule and enqueue automatic sync work when a mirror becomes due.

#### Scenario: Enqueue scheduled sync for a due mirror
- **WHEN** the scheduler evaluates an enabled mirror whose stored cron schedule is due
- **THEN** the system enqueues a full-sync job for that mirror in the shared sync job queue
- **AND** the scheduled job uses the same sync scope as the manual sync action

### Requirement: Scheduler avoids redundant active work
The system SHALL avoid enqueueing a new scheduled sync when the same mirror already has active queued or running work.

#### Scenario: Skip enqueue when mirror already has active work
- **WHEN** the scheduler evaluates a due mirror that already has a queued or running sync job
- **THEN** the system does not enqueue another scheduled sync job for that mirror

### Requirement: Scheduler state survives process restarts
The system SHALL persist enough scheduler state to avoid repeatedly enqueueing the same due mirror across polling cycles and restarts.

#### Scenario: Restart scheduler after a due evaluation
- **WHEN** the scheduler process restarts after evaluating mirror schedules
- **THEN** the system continues evaluating mirrors from persisted scheduling state
- **AND** it does not repeatedly enqueue duplicate jobs for the same already-recorded due window

### Requirement: Scheduled sync uses a documented timezone
The system SHALL evaluate stored cron schedules using a single documented runtime timezone.

#### Scenario: User reviews schedule behavior
- **WHEN** an authenticated user configures or reviews a mirror cron schedule
- **THEN** the system identifies the timezone used for schedule evaluation in the product UI or documentation
