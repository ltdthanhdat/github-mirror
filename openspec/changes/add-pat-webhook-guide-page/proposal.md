## Why

Users currently have no in-app guidance for the two setup steps that are easiest to get wrong: creating GitHub personal access tokens with the right permissions and configuring the source repository webhook after the mirror is created. The current UI assumes that users already know both flows, which creates avoidable friction during first-time setup.

## What Changes

- Add an authenticated guide page that explains how to create the source and target GitHub PAT tokens needed by the mirror form.
- Document the recommended token access levels and operational notes, such as one-time token visibility and possible organization approval requirements.
- Explain how to add the GitHub webhook on the source repository after a mirror is created, including the payload URL, content type, and push event selection.
- Add navigation entry points so users can discover the guide from the app UI while creating or reviewing a mirror.

## Capabilities

### New Capabilities
- `mirror-setup-guide`: An authenticated in-app guide for PAT token creation and GitHub webhook setup used by the mirror workflow.

### Modified Capabilities
- None.

## Impact

- Affected Go UI routing and handler wiring for the new authenticated guide page.
- Affected HTML templates and shared navigation so the guide is reachable from the existing UI.
- New OpenSpec requirements for guide accessibility and setup content.
