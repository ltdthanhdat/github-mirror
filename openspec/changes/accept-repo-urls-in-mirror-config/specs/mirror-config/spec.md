## MODIFIED Requirements

### Requirement: Users can create mirror configurations
The system SHALL allow authenticated users to define a mirror relationship between a source and target GitHub repository.

#### Scenario: Create mirror configuration from repository URLs
- **WHEN** an authenticated user submits the mirror creation form with valid `source_url`, `target_url`, and tokens
- **THEN** the system parses the GitHub repository URLs into owner and repository values
- **AND** the system stores the mirror configuration with encrypted tokens and normalized clone URLs
- **AND** the system performs an initial sync to mirror all refs from source to target

#### Scenario: Reject unsupported repository URLs
- **WHEN** an authenticated user submits a mirror creation request with a source or target URL that is not a valid GitHub repository root URL
- **THEN** the system rejects the request as invalid
- **AND** the system does not create a mirror configuration
