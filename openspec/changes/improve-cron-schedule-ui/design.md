## Context

The current product already persists `sync_schedule`, validates cron expressions, and displays the configured schedule on the mirror detail page. However, the only editing surface is the full mirror edit form, which mixes repository URLs, tokens, branch options, and schedule settings into one workflow. Users who arrive on the detail page can see the schedule value but have no direct UI path dedicated to changing or clearing it.

The existing `UpdateMirrorHandler` expects the full mirror form payload, so the current HTTP contract is not a good fit for a small schedule-only interaction. The UI already has separate detail and form handlers, and route additions are low risk.

## Goals / Non-Goals

**Goals:**
- Add a discoverable schedule management action from the mirror detail page.
- Let users update or clear the cron schedule without editing unrelated mirror settings.
- Reuse the existing cron validation rules and UTC messaging.
- Keep the existing full mirror edit form working.

**Non-Goals:**
- Changing scheduler runtime behavior, polling cadence, or queue semantics.
- Changing the database schema or persisted schedule fields.
- Reworking the entire mirror detail page layout beyond what is needed for schedule management.

## Decisions

### Add a dedicated schedule edit route and form
- **Decision**: Introduce a dedicated UI route to render a schedule-specific form and a matching POST endpoint to save or clear the cron schedule.
- **Why**: The dedicated flow makes the schedule feature visible from the detail page and avoids forcing users through unrelated fields.
- **Alternatives considered**:
  - Reuse the existing full edit form only: rejected because the feature remains hard to discover.
  - Add inline HTMX editing inside the configuration table: rejected for now because it increases template complexity and form-state handling with little functional benefit over a focused page.

### Reuse existing schedule validation and persistence rules
- **Decision**: The schedule-specific save handler will call the same cron validation helper and persist through `MirrorStore.UpdateMirrorConfig`.
- **Why**: This keeps schedule semantics consistent across create, full edit, and schedule-only edit flows.
- **Alternatives considered**:
  - Create a separate validation path: rejected because it duplicates rules and increases drift risk.

### Redirect back to mirror detail with focused feedback
- **Decision**: After a successful save or clear action, return the user to the mirror detail page with a flash message. Validation failures re-render the schedule form with the submitted value and an inline error.
- **Why**: The detail page is where users review the effect of the change, and this matches the existing UI pattern used by the broader mirror edit flow.
- **Alternatives considered**:
  - Stay on the schedule form after success: rejected because it adds an extra navigation step to confirm the result.

## Risks / Trade-offs

- [Two edit paths can drift] → Mitigation: keep schedule validation in the shared helper and cover both flows with handler/UI tests.
- [Users may still use the full edit form for schedule changes] → Mitigation: keep both flows functional and consistent rather than trying to force one path.
- [Extra route adds small maintenance overhead] → Mitigation: keep the new form narrowly scoped to one field and reuse existing rendering patterns.
