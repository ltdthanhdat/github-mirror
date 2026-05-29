-- Create mirror_configs table
CREATE TABLE mirror_configs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    name TEXT NOT NULL,

    source_owner TEXT NOT NULL,
    source_repo TEXT NOT NULL,
    source_repo_url TEXT NOT NULL,

    target_owner TEXT NOT NULL,
    target_repo TEXT NOT NULL,
    target_repo_url TEXT NOT NULL,

    source_token_enc TEXT NOT NULL,
    target_token_enc TEXT NOT NULL,
    webhook_secret_enc TEXT NOT NULL,

    branch_pattern TEXT NOT NULL DEFAULT '*',
    sync_tags BOOLEAN NOT NULL DEFAULT TRUE,
    sync_deletes BOOLEAN NOT NULL DEFAULT FALSE,
    allow_force_update BOOLEAN NOT NULL DEFAULT TRUE,

    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_synced_at TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
