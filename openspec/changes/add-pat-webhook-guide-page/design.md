## Context

The current application has authenticated UI pages for the dashboard, mirror creation form, and mirror detail view, but no in-app documentation page for setup prerequisites. Users have to leave the app or guess how to create GitHub PAT tokens and where to install the webhook, even though those steps are required to make the mirror flow succeed. The existing UI architecture is simple: authenticated GET routes render server-side HTML templates through `internal/ui.Handler`, and shared navigation comes from the base layout template.

## Goals / Non-Goals

**Goals:**
- Add a lightweight authenticated guide page within the existing server-rendered UI.
- Explain two setup flows clearly: creating the source and target PAT tokens, and adding the GitHub webhook on the source repository.
- Make the guide discoverable from the existing UI without changing the core mirror creation flow.

**Non-Goals:**
- Supporting non-GitHub providers or documenting generic git hosting flows.
- Adding live PAT validation, token generation, or webhook delivery diagnostics to the guide page.
- Embedding screenshots, interactive tours, or a larger documentation system.

## Decisions

### Add a dedicated authenticated page instead of expanding existing screens
- **Decision:** Introduce a standalone guide page reachable from shared navigation and targeted CTAs.
- **Why:** The setup guidance spans multiple steps before and after mirror creation. Forcing all of that content into the create form or detail page would make those pages harder to scan and maintain.
- **Alternative considered:** Add inline help blocks directly to `mirror_form.html` and `mirror_detail.html`. Rejected because it would fragment the instructions across multiple pages and duplicate content.

### Keep the guide content static and server-rendered
- **Decision:** Render a static HTML template through the existing `internal/ui.Handler`.
- **Why:** The content does not depend on per-user data beyond requiring authentication, and the current UI already uses server-rendered templates without a client-side state layer.
- **Alternative considered:** Store guide content in markdown or fetch it dynamically. Rejected because it adds moving parts without solving a current problem.

### Reuse the existing authenticated route group
- **Decision:** Serve the guide page inside the current protected router group.
- **Why:** The page is directly tied to the authenticated mirror workflow, and keeping it protected avoids introducing a separate public content surface or duplicate layout handling.
- **Alternative considered:** Make the page public. Rejected because the current app navigation and support flow are centered on signed-in users working inside the dashboard.

## Risks / Trade-offs

- [GitHub PAT UX changes over time] -> Keep the copy focused on stable navigation labels and permission intent rather than brittle UI micro-details.
- [Users may assume the guide is exhaustive product documentation] -> Keep the page tightly scoped to PAT creation and webhook setup, with clear CTA paths back to the actual mirror workflow.
- [Shared navigation grows denser] -> Add only a single guide entry and avoid introducing a broader docs taxonomy in this change.
