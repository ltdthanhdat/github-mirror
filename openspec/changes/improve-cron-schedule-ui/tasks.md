## 1. Add dedicated schedule management flow

- [x] 1.1 Add a mirror detail UI action that takes users to a dedicated cron schedule management screen.
- [x] 1.2 Add a schedule-specific UI handler, route, and template that render the current schedule, UTC guidance, and a save/clear form.
- [x] 1.3 Add a schedule-specific update handler that validates the submitted cron expression and persists only the schedule field.

## 2. Keep schedule behavior consistent

- [x] 2.1 Reuse the existing cron validation helper and mirror store update path so schedule-only edits follow the same rules as create and full edit.
- [x] 2.2 Return clear success feedback when a schedule is saved or cleared, and re-render the dedicated form with inline errors when validation fails.

## 3. Verify the new UI flow

- [x] 3.1 Add handler tests covering schedule form access, valid save, clear schedule, and invalid cron rejection without changing other mirror fields.
- [x] 3.2 Add UI/template coverage ensuring the mirror detail page exposes the schedule management action and the dedicated form shows UTC guidance and the current value.
