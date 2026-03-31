-- Add read_at column to entries for read/unread tracking.
ALTER TABLE entries ADD COLUMN read_at TEXT;

-- Add HTTP cache headers to feeds for conditional GET support.
ALTER TABLE feeds ADD COLUMN etag TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN last_modified TEXT NOT NULL DEFAULT '';
