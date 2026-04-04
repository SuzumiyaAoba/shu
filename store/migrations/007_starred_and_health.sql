-- +goose Up
-- Bookmark/star support for entries.
ALTER TABLE entries ADD COLUMN starred_at TEXT;

-- Feed health monitoring: track consecutive failures and last error.
ALTER TABLE feeds ADD COLUMN error_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE feeds ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN disabled INTEGER NOT NULL DEFAULT 0;

-- Per-feed fetch interval override (seconds). 0 means use global default.
ALTER TABLE feeds ADD COLUMN fetch_interval_sec INTEGER NOT NULL DEFAULT 0;

-- +goose Down
SELECT 1;
