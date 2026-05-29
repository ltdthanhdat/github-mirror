-- Indexes for performance

-- Users
CREATE INDEX idx_users_email ON users(email);

-- Mirror configs
CREATE INDEX idx_mirror_configs_user_id ON mirror_configs(user_id);

-- Sync jobs
CREATE INDEX idx_sync_jobs_mirror_config_id ON sync_jobs(mirror_config_id);
CREATE INDEX idx_sync_jobs_status ON sync_jobs(status);
CREATE INDEX idx_sync_jobs_created_at ON sync_jobs(created_at);
