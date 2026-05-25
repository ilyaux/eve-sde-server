-- +goose Up
ALTER TABLE api_keys ADD COLUMN key_hash TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE key_hash IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_api_keys_key_hash;
