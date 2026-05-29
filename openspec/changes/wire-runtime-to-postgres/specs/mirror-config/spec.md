## MODIFIED Requirements

### Requirement: Users can create mirror configurations
The system SHALL allow authenticated users to define a mirror relationship between a source and target GitHub repository.

#### Scenario: Create mirror configuration
- **WHEN** an authenticated user submits the mirror creation form with valid source owner, source repo, target owner, target repo, and tokens
- **THEN** the system stores the mirror configuration durably in PostgreSQL with encrypted tokens and returns the created configuration
- **AND** the system performs an initial sync to mirror all refs from source to target

### Requirement: Users can update mirror configurations
The system SHALL allow authenticated users to update existing mirror configurations.

#### Scenario: Update mirror configuration
- **WHEN** an authenticated user submits updates to a mirror configuration they own
- **THEN** the system persists the updated configuration durably in PostgreSQL and encrypts any new tokens
- **AND** the system validates that the new configuration is valid before saving

### Requirement: Users can delete mirror configurations
The system SHALL allow authenticated users to delete mirror configurations they own.

#### Scenario: Delete mirror configuration
- **WHEN** an authenticated user requests deletion of a mirror configuration they own
- **THEN** the system removes the persisted configuration and associated sync job data
- **AND** the system cleans up the associated Git cache directory
- **AND** the system returns success confirmation

## ADDED Requirements

### Requirement: Mirror configurations SHALL remain available across restarts
The system SHALL preserve each user's mirror configurations across service restarts when the server reconnects to the same PostgreSQL database.

#### Scenario: List mirrors after restart
- **WHEN** a user created one or more mirror configurations before the service restarts
- **THEN** the system returns those same mirror configurations after restart
- **AND** the user can still view and edit them without recreating the records
