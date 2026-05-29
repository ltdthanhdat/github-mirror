-- Create sync_jobs table
CREATE TABLE sync_jobs (
    id SERIAL PRIMARY KEY,
    mirror_config_id INTEGER NOT NULL REFERENCES mirror_configs(id) ON DELETE CASCADE,

    github_delivery_id TEXT,
    ref TEXT NOT NULL,
    ref_type TEXT NOT NULL,
    branch_or_tag TEXT NOT NULL,
    after_sha TEXT,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,

    status TEXT NOT NULL DEFAULT 'queued',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,

    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(mirror_config_id, github_delivery_id)
);
