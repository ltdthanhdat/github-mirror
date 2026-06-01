## Why

Per-mirror cron scheduling already exists, but it is easy to miss because the schedule is only editable inside the general mirror edit form. Operators reviewing a mirror configuration from the detail page cannot quickly discover how to change or clear the schedule, which makes the feature look incomplete.

## What Changes

- Add a dedicated schedule management entry point from the mirror detail page.
- Add a focused UI for updating or clearing a mirror's cron schedule without editing unrelated mirror settings.
- Preserve the existing cron validation rules and UTC schedule messaging in the new flow.
- Show success and validation feedback in the schedule editing flow so users can confirm the saved value.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `mirror-config`: extend mirror detail and edit behavior so users can directly manage the cron schedule from a dedicated UI flow.

## Impact

- Affected code: UI handlers, templates, mirror update handlers, and related tests.
- No new backend scheduler behavior, database schema, or external dependencies.
- Existing mirror update API remains the persistence path for the cron schedule.
