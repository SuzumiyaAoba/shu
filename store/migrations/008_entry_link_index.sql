-- Index on entry link for cross-feed duplicate detection.
CREATE INDEX IF NOT EXISTS idx_entries_link ON entries(link);
