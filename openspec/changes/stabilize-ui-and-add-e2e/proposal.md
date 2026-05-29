## Why

The current mirror UI is not in a releasable state: protected pages return runtime errors, key HTMX form actions do not match backend contracts, and there is no automated end-to-end coverage to catch these regressions. We need to stabilize the UI flows first, then add executable regression checks so UI and UX improvements can be made with confidence.

## What Changes

- Fix runtime rendering issues for authenticated UI pages so the dashboard, mirror creation page, and mirror detail page load successfully.
- Align UI actions with backend behavior for the primary mirror management flows, including create, view, sync/test actions, and delete.
- Add automated end-to-end coverage for the core authenticated flows and capture visual evidence from the running app during verification.
- Improve the base UI presentation for the main pages after the flows are stable, focusing on readability, feedback states, and mobile-safe layout rather than expanding scope.

## Capabilities

### New Capabilities
- `ui-regression-coverage`: End-to-end verification for the authenticated mirror management UI, including executable browser flows and captured visual evidence for review.

### Modified Capabilities
- `mirror-config`: Mirror configuration management requirements will be tightened so the authenticated UI pages and actions are required to load and complete successfully through the browser-based user flows.

## Impact

- Affected Go HTTP/UI code in `internal/http`, `internal/ui`, and HTML templates under `internal/ui/templates`
- New automated browser test setup and test files for regression coverage
- Verification workflow will include runtime screenshots for UI review before final polish decisions
