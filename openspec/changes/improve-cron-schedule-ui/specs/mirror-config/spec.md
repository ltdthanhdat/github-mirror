## MODIFIED Requirements

### Requirement: Users can view mirror configuration details
The system SHALL allow authenticated users to view detailed information about a specific mirror configuration.

#### Scenario: View mirror configuration details
- **WHEN** an authenticated user requests details for a specific mirror configuration ID
- **THEN** the system returns the configuration if it belongs to the user
- **AND** the response includes webhook URL and setup instructions
- **AND** token fields are masked or omitted for security

#### Scenario: Access schedule management from mirror details
- **WHEN** an authenticated user views a mirror configuration detail page
- **THEN** the system shows the current cron schedule state for that mirror
- **AND** the system provides a dedicated UI action to change or clear the schedule
- **AND** the UI identifies that cron schedules are evaluated in UTC

### Requirement: Users can update mirror configurations
The system SHALL allow authenticated users to update existing mirror configurations.

#### Scenario: Update mirror configuration
- **WHEN** an authenticated user submits updates to a mirror configuration they own
- **THEN** the system updates the configuration and encrypts any new tokens
- **AND** the system validates that the new configuration is valid before saving

#### Scenario: Update mirror schedule from dedicated UI
- **WHEN** an authenticated user submits an empty or valid cron schedule from the dedicated schedule management UI
- **THEN** the system updates only the mirror's persisted schedule value
- **AND** the system preserves the other mirror configuration fields unchanged
- **AND** the response confirms whether automatic sync is configured or cleared

#### Scenario: Reject invalid schedule from dedicated UI
- **WHEN** an authenticated user submits an invalid cron schedule from the dedicated schedule management UI
- **THEN** the system rejects the request
- **AND** the existing mirror configuration remains unchanged
- **AND** the UI re-renders the submitted value with an explanation that the cron schedule is invalid
