## ADDED Requirements

### Requirement: System provides automated browser regression coverage for the primary mirror UI
The system SHALL include executable end-to-end coverage for the authenticated dashboard, mirror creation flow, and mirror detail flow so regressions in routes, templates, auth wiring, and HTMX behavior are caught before release.

#### Scenario: Run primary UI browser regression
- **WHEN** the automated browser regression suite is executed against the application
- **THEN** it verifies that an authenticated user can load the dashboard, open the mirror creation page, create a mirror, and open the resulting detail page
- **AND** the suite fails if any covered page returns a runtime error, broken route, or failed user action

### Requirement: System captures visual evidence for reviewed UI states
The system SHALL capture screenshots for the primary reviewed UI states after the regression flow passes so UI/UX changes can be assessed against the rendered application.

#### Scenario: Capture review screenshots
- **WHEN** the verified browser flow reaches the dashboard, mirror creation page, or mirror detail page
- **THEN** the test or verification workflow captures screenshots for those pages
- **AND** the resulting screenshots are available for manual UI review before concluding the change
