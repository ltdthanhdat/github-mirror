## 1. Setup and Foundations

- [x] 1.1 Initialize Go module and basic directory structure
- [x] 1.2 Create Dockerfile and docker-compose.yml for postgres
- [x] 1.3 Implement basic HTTP router with chi or stdlib
- [x] 1.4 Add basic auth middleware (placeholder)
- [x] 1.5 Create initial database schema migration (users table)

## 2. User Authentication and Model

- [x] 2.1 Define User model and methods
- [x] 2.2 Implement password hashing (bcrypt)
- [x] 2.3 Create user registration and login endpoints (API)
- [x] 2.4 Protect API routes with basic auth middleware
- [x] 2.5 Add user context to request (from basic auth)

## 3. Database Schema for Mirror Service

- [x] 3.1 Create mirror_configs table migration
- [x] 3.2 Create sync_jobs table migration (with SKIP LOCKED capable columns)
- [x] 3.3 Implement Store interface for mirror_configs and sync_jobs
- [x] 3.4 Add database indexes for performance (user_id, status, etc.)

## 4. Token Encryption

- [x] 4.1 Design encryption module (AES-GCM with APP_ENCRYPTION_KEY)
- [x] 4.2 Implement encrypt and decrypt functions
- [x] 4.3 Add encryption fields to mirror_configs model (source_token_enc, etc.)
- [x] 4.4 Ensure tokens are encrypted before DB write and decrypted after read

## 5. Webhook Receiver

- [x] 5.1 Create webhook handler route (/webhooks/github/:id)
- [x] 5.2 Implement GitHub signature verification (X-Hub-Signature-256)
- [x] 5.3 Parse push payload and extract ref information
- [x] 5.4 Check allowed refs based on mirror config (branch_pattern, sync_tags, sync_deletes)
- [x] 5.5 Enqueue sync job with idempotency (using X-GitHub-Delivery)
- [x] 5.6 Return appropriate HTTP status (202 Accepted)

## 6. Job Queue Mechanism

- [x] 6.1 Implement job claiming query using SKIP LOCKED
- [x] 6.2 Create SyncJob model and methods
- [x] 6.3 Add job status transitions (queued -> running -> succeeded/failed)
- [x] 6.4 Implement retry logic with attempts counter
- [x] 6.5 Store last error on failure

## 7. Sync Worker

- [x] 7.1 Create worker loop that polls for jobs
- [x] 7.2 Implement bare Git repository cache per mirror config
- [x] 7.3 Add file-based locking (flock) per mirror config
- [x] 7.4 Implement git fetch and push logic for branches
- [x] 7.5 Implement tag syncing if sync_tags enabled
- [x] 7.6 Implement branch deletion if sync_deletes enabled
- [x] 7.6 Handle force push based on allow_force_update flag
- [x] 7.7 Add timeout for git commands
- [x] 7.8 Clean up temporary workspaces (if any) after job
- [x] 7.9 Update job status to succeeded or failed based on git operation

## 8. Mirror Config CRUD API

- [x] 8.1 GET /mirrors - list user's mirror configurations
- [x] 8.2 GET /mirrors/new - render form for creating mirror (UI)
- [x] 8.3 POST /mirrors - create new mirror configuration
- [x] 8.4 GET /mirrors/:id - show mirror configuration detail
- [x] 8.5 POST /mirrors/:id/test - test source and target tokens
- [x] 8.6 POST /mirrors/:id/retry - retry a failed job
- [x] 8.7 POST /mirrors/:id/sync - trigger manual sync
- [x] 8.8 DELETE /mirrors/:id - delete mirror configuration

## 9. HTMX UI Components

- [x] 9.1 Create base layout template
- [x] 9.2 Implement dashboard page listing mirrors
- [x] 9.3 Implement mirror creation form (HTMX enhanced)
- [x] 9.4 Implement mirror detail page showing webhook URL and setup instructions
- [x] 9.5 Add HTMX endpoints for form submission and dynamic updates
- [x] 9.6 Show sync status and last synced time on dashboard
- [x] 9.7 Add buttons for retry and manual sync on detail page

## 10. Initial Sync and Logging

- [x] 10.1 Implement initial sync when creating a new mirror (full mirror)
- [x] 10.2 Add structured logging (JSON or logfmt) with request IDs
- [x] 10.3 Mask tokens in any logged git URLs
- [ ] 10.4 Implement log retention cleanup (delete logs older than 7 days)
- [ ] 10.5 Add periodic cleanup job for expired logs and stale git caches

## 11. Observability and Reliability

- [x] 11.1 Add health check endpoint
- [ ] 11.2 Implement metrics for job queue depth and sync duration (optional)
- [x] 11.3 Add graceful shutdown for worker (finish current job)
- [x] 11.4 Ensure proper error handling and return meaningful error messages

## 12. Future Enhancements (Post-MVP)

- [x] 12.1 Research and plan for GitHub App integration (to replace PAT)
- [x] 12.2 Consider implementing pagination for mirror lists
- [x] 12.3 Add email notifications for failed syncs
- [x] 12.4 Implement webhook secret rotation UI