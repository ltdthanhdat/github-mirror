## Context

The current service has a server-rendered HTMX UI, but the authenticated pages are not operational at runtime. Template execution fails for the base layout, form submissions are wired as browser form posts while the handler expects JSON, and the delete action points to a route that does not exist. There is also no automated browser coverage, so these regressions were not caught during development.

This change spans HTTP handlers, UI templates, and a new test harness. It also introduces a tighter verification loop: stabilize the flows first, then capture screenshots from the running app to drive UI polish decisions.

## Goals / Non-Goals

**Goals:**
- Restore the dashboard, new mirror page, and mirror detail page so they render successfully for an authenticated user.
- Align HTMX/browser submissions with backend request parsing and route definitions for the core mirror management flows.
- Add executable end-to-end regression checks for the primary browser flows.
- Leave enough room for one intentional UI polish pass after the flows and tests are stable.

**Non-Goals:**
- Replacing Basic Auth with a dedicated login UI or session system
- Reworking the worker, webhook, or data model architecture
- Building a comprehensive visual design system
- Adding broad coverage for optional or future features outside the main mirror management path

## Decisions

### 1. Fix the runtime UI path before adding polish
- **Why**: The current blocking issue is correctness, not aesthetics. A page that returns `504` cannot be meaningfully reviewed for UX.
- **Alternatives considered**: Polish templates first; add tests against the current broken behavior.
- **Trade-off**: UI improvements come slightly later in the sequence, but they are evaluated on a working app instead of a broken baseline.

### 2. Make the primary create/delete flows browser-native instead of JSON-only
- **Why**: The existing UI already uses HTMX form/button submissions. It is simpler and lower-risk to let handlers accept standard form data or HTMX-compatible requests than to retrofit custom client-side JSON serialization.
- **Alternatives considered**: Rewrite the UI to submit JSON with client-side JavaScript; split separate UI-only endpoints.
- **Trade-off**: Handlers may need to support both JSON API and form submissions, but that preserves the existing API shape while making the UI actually work.

### 3. Keep route and template changes surgical
- **Why**: The user asked for stabilization, tests, and UI review, not a redesign of the whole app. Small contract fixes reduce regression risk.
- **Alternatives considered**: Refactor the entire UI routing layer or template system.
- **Trade-off**: Some rough edges may remain outside the covered flows, but the change stays tightly scoped and easier to verify.

### 4. Add a lightweight browser e2e harness around the running Go server
- **Why**: The critical failures are integration failures across auth, routes, templates, and HTMX behavior. Unit tests alone will miss them.
- **Alternatives considered**: Only `httptest` integration tests; manual browser checks without automation.
- **Trade-off**: Browser tests add setup cost, but they provide durable regression coverage for the exact flows that already broke.

### 5. Use screenshots as a required review artifact after the flow is stable
- **Why**: The user specifically wants a loop of testing, screenshots, and UI/UX improvement. Captured pages create a concrete review surface.
- **Alternatives considered**: Rely only on DOM assertions or code review.
- **Trade-off**: Screenshot capture adds one more verification step, but it keeps UI changes grounded in actual rendered output.

## Risks / Trade-offs

- **[Dual request parsing paths drift]** → Mitigation: keep the accepted fields identical for JSON and form submissions, and cover both the UI path and key handler behavior in tests.
- **[E2E tests become flaky]** → Mitigation: run the app in-process or in a controlled local test harness, avoid network dependencies to external GitHub calls in the covered flows, and prefer deterministic seed data.
- **[UI polish expands scope]** → Mitigation: constrain polish to layout clarity, action feedback, and responsive readability on the existing pages only.
- **[Screenshot review misses functional regressions]** → Mitigation: use screenshots only after functional e2e assertions pass; they complement, not replace, automated checks.

## Migration Plan

1. Stabilize template loading and authenticated page rendering.
2. Align create/delete/UI action contracts so the main flows work in a browser.
3. Add end-to-end coverage for dashboard, creation, detail, and delete/sync action surfaces.
4. Capture screenshots from the running app and perform one focused UI/UX improvement pass.
5. Re-run tests and refresh screenshots to confirm the final state.

Rollback is straightforward because the change is localized to app code and test assets; reverting the change restores the previous behavior, though that previous behavior is known to be broken.

## Open Questions

- Which browser test tool fits best with the current repo constraints: Go-driven browser automation or a small Playwright layer?
- Should token test/sync actions be exercised with mocked responses in e2e, or should the initial regression scope stop at page load and form flows?
