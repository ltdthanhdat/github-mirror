## Why

Creating a mirror currently requires users to manually split each GitHub repository into owner and repository fields. That adds friction to a workflow where users usually already have the source and target repository URLs and expect to paste them directly.

## What Changes

- Update mirror creation inputs to accept `source_url` and `target_url` instead of separate owner and repository fields.
- Parse and validate GitHub repository URLs on the server, then derive owner, repository, and normalized clone URLs from those inputs.
- Update the mirror creation UI, API contract, and tests to reflect the URL-based input flow.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `mirror-config`: mirror creation changes from owner/repository field submission to GitHub repository URL submission, including validation and normalization rules.

## Impact

- Affected UI template for mirror creation.
- Affected mirror creation handler and request validation.
- Affected mirror configuration specs and end-to-end tests.
- No database schema change is required if existing derived storage fields remain in use.
