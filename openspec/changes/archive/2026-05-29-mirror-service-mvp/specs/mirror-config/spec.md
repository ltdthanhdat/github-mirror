## ADDED Requirements

### Requirement: Users can create mirror configurations
The system SHALL allow authenticated users to define a mirror relationship between a source and target GitHub repository.

#### Scenario: Create mirror configuration
- **WHEN** an authenticated user submits the mirror creation form with valid source owner, source repo, target owner, target repo, and tokens
- **THEN** the system stores the mirror configuration with encrypted tokens and returns the created configuration
- **AND** the system performs an initial sync to mirror all refs from source to target

### Requirement: Users can list their mirror configurations
The system SHALL allow authenticated users to view a list of their mirror configurations.

#### Scenario: List mirror configurations
- **WHEN** an authenticated user requests the list of mirrors
- **THEN** the system returns all mirror configurations belonging to that user
- **AND** each configuration includes masked token information and last sync status

### Requirement: Users can view mirror configuration details
The system SHALL allow authenticated users to view detailed information about a specific mirror configuration.

#### Scenario: View mirror configuration details
- **WHEN** an authenticated user requests details for a specific mirror configuration ID
- **THEN** the system returns the configuration if it belongs to the user
- **AND** the response includes webhook URL and setup instructions
- **AND** token fields are masked or omitted for security

### Requirement: Users can update mirror configurations
The system SHALL allow authenticated users to update existing mirror configurations.

#### Scenario: Update mirror configuration
- **WHEN** an authenticated user submits updates to a mirror configuration they own
- **THEN** the system updates the configuration and encrypts any new tokens
- **AND** the system validates that the new configuration is valid before saving

### Requirement: Users can delete mirror configurations
The system SHALL allow authenticated users to delete mirror configurations they own.

#### Scenario: Delete mirror configuration
- **WHEN** an authenticated user requests deletion of a mirror configuration they own
- **THEN** the system removes the configuration and associated data
- **AND** the system cleans up the associated Git cache directory
- **AND** the system returns success confirmation

### Requirement: System validates mirror configuration ownership
The system SHALL ensure that users can only access mirror configurations they own.

#### Scenario: Prevent unauthorized access
- **WHEN** a user attempts to access a mirror configuration owned by another user
- **THEN** the system returns a 404 or 403 error
- **AND** no information about the configuration is leaked