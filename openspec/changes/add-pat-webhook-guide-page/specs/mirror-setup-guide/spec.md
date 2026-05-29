## ADDED Requirements

### Requirement: Authenticated users can access a mirror setup guide
The system SHALL provide an authenticated guide page that explains how to prepare PAT tokens and the GitHub webhook required by the mirror workflow.

#### Scenario: Open the guide from the authenticated UI
- **WHEN** an authenticated user navigates to the guide page from the application UI
- **THEN** the system returns a rendered guide page within the standard authenticated layout
- **AND** the page is reachable from at least one primary navigation entry point in the authenticated experience

### Requirement: Guide explains GitHub PAT token setup
The system SHALL explain how users obtain and use the source and target GitHub PAT tokens required by mirror creation.

#### Scenario: Review PAT instructions
- **WHEN** an authenticated user reads the guide page
- **THEN** the page explains where to create GitHub personal access tokens
- **AND** the page distinguishes the source token from the target token
- **AND** the page states that the source token needs repository read access and the target token needs repository write access

### Requirement: Guide explains GitHub webhook setup
The system SHALL explain how users configure the source repository webhook after creating a mirror.

#### Scenario: Review webhook instructions
- **WHEN** an authenticated user reads the guide page
- **THEN** the page states that the webhook must be added on the source repository
- **AND** the page explains that the payload URL comes from the mirror detail page
- **AND** the page instructs the user to use `application/json` content type and the `Push` event

### Requirement: Guide stays scoped to supported setup behavior
The system SHALL present setup guidance that matches the supported mirror workflow and omits unsupported or speculative instructions.

#### Scenario: Read supported setup guidance
- **WHEN** an authenticated user reads the guide page
- **THEN** the page focuses on GitHub PAT creation and GitHub webhook setup for the mirror workflow
- **AND** the page does not include unrelated troubleshooting or extra setup sections outside that scope
