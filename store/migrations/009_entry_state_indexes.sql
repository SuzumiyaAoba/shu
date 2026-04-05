-- +goose Up
-- Partial indexes for common entry state filters.
CREATE INDEX IF NOT EXISTS idx_entries_unread ON entries(fetched_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_entries_starred ON entries(starred_at DESC)
    WHERE starred_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_entries_starred;
DROP INDEX IF EXISTS idx_entries_unread;
