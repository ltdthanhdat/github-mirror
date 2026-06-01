ALTER TABLE mirror_configs
    ADD COLUMN sync_schedule TEXT,
    ADD COLUMN last_scheduled_at TIMESTAMPTZ;

CREATE INDEX idx_mirror_configs_sync_schedule
    ON mirror_configs(enabled, sync_schedule)
    WHERE enabled = TRUE AND sync_schedule IS NOT NULL AND sync_schedule <> '';
