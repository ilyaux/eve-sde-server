-- +goose Up
-- Table for tracking SDE versions and updates
CREATE TABLE IF NOT EXISTS sde_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    checksum TEXT NOT NULL UNIQUE,
    downloaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    imported_at TIMESTAMP,
    import_duration_seconds INTEGER,
    items_count INTEGER,
    error TEXT
);

-- Index for quick checksum lookup
CREATE INDEX IF NOT EXISTS idx_sde_versions_checksum ON sde_versions(checksum);
CREATE INDEX IF NOT EXISTS idx_sde_versions_downloaded_at ON sde_versions(downloaded_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_sde_versions_downloaded_at;
DROP INDEX IF EXISTS idx_sde_versions_checksum;
DROP TABLE IF EXISTS sde_versions;
