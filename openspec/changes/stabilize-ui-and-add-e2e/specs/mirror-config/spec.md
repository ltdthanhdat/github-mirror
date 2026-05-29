## MODIFIED Requirements

### Requirement: Users can create mirror configurations
The system SHALL allow authenticated users to create a mirror relationship from the browser UI and through the existing authenticated API using the same required mirror fields.

#### Scenario: Create mirror configuration from the authenticated UI
- **WHEN** an authenticated user submits the mirror creation form with valid source owner, source repo, target owner, target repo, and tokens
- **THEN** the system stores the mirror configuration and returns a successful browser response without a server error
- **AND** the response updates the UI with confirmation or the created mirror view
- **AND** the system performs an initial sync to mirror all refs from source to target

#### Scenario: Create mirror configuration from the API
- **WHEN** an authenticated user sends a valid API request to create a mirror configuration
- **THEN** the system stores the mirror configuration with the same required fields used by the UI flow
- **AND** the system performs an initial sync to mirror all refs from source to target

### Requirement: Users can list their mirror configurations
The system SHALL allow authenticated users to open the dashboard successfully and view a list of their mirror configurations.

#### Scenario: Dashboard renders for an authenticated user
- **WHEN** an authenticated user requests the dashboard page
- **THEN** the system returns a successful HTML response
- **AND** the page renders the mirror list or an empty state without template or runtime errors

### Requirement: Users can view mirror configuration details
The system SHALL allow authenticated users to open a mirror detail page that renders configuration details and available mirror actions successfully.

#### Scenario: View mirror configuration details in the browser
- **WHEN** an authenticated user requests details for a specific mirror configuration ID they own
- **THEN** the system returns a successful HTML response
- **AND** the page renders configuration details, webhook setup instructions, and available mirror actions without template or runtime errors
- **AND** token fields are masked or omitted for security

### Requirement: Users can delete mirror configurations
The system SHALL allow authenticated users to delete mirror configurations they own from the authenticated UI and receive a successful browser response.

#### Scenario: Delete mirror configuration from the UI
- **WHEN** an authenticated user confirms deletion for a mirror configuration they own from the detail page
- **THEN** the system removes the configuration and associated data
- **AND** the browser flow completes through a valid application route
- **AND** the UI returns success confirmation or navigates back to a state where the deleted mirror is no longer shown
