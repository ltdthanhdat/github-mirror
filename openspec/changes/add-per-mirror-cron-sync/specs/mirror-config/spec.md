## MODIFIED Requirements

### Requirement: Users can create mirror configurations
The system SHALL allow authenticated users to define a mirror relationship between a source and target GitHub repository, including an optional per-mirror cron schedule for automatic sync.

#### Scenario: Create mirror configuration
- **WHEN** an authenticated user submits the mirror creation form with valid source owner, source repo, target owner, target repo, tokens, and an empty or valid cron schedule
- **THEN** the system stores the mirror configuration with encrypted tokens and the provided schedule value
- **AND** the system performs an initial sync to mirror all refs from source to target

#### Scenario: Reject invalid cron schedule during creation
- **WHEN** an authenticated user submits the mirror creation form with an invalid cron schedule
- **THEN** the system rejects the request
- **AND** the system does not create the mirror configuration
- **AND** the response explains that the cron schedule is invalid

### Requirement: Users can view mirror configuration details
The system SHALL allow authenticated users to view detailed information about a specific mirror configuration, including whether automatic scheduled sync is configured.

#### Scenario: View mirror configuration details
- **WHEN** an authenticated user requests details for a specific mirror configuration ID
- **THEN** the system returns the configuration if it belongs to the user
- **AND** the response includes webhook URL and setup instructions
- **AND** the response includes the configured cron schedule when automatic sync is enabled
- **AND** token fields are masked or omitted for security

### Requirement: Users can update mirror configurations
The system SHALL allow authenticated users to update existing mirror configurations, including changing or clearing the per-mirror cron schedule.

#### Scenario: Update mirror configuration
- **WHEN** an authenticated user submits updates to a mirror configuration they own with an empty or valid cron schedule
- **THEN** the system updates the configuration and encrypts any new tokens
- **AND** the system validates that the new configuration is valid before saving
- **AND** the updated schedule value is persisted

#### Scenario: Reject invalid cron schedule during update
- **WHEN** an authenticated user submits updates containing an invalid cron schedule
- **THEN** the system rejects the request
- **AND** the existing mirror configuration remains unchanged
- **AND** the response explains that the cron schedule is invalid
