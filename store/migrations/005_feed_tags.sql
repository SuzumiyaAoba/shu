-- +goose Up
-- Tag management for feeds.
CREATE TABLE IF NOT EXISTS tags (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS feed_tags (
    feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (feed_id, tag_id)
);

CREATE INDEX idx_feed_tags_tag_id ON feed_tags(tag_id);

-- +goose Down
DROP INDEX IF EXISTS idx_feed_tags_tag_id;
DROP TABLE IF EXISTS feed_tags;
DROP TABLE IF EXISTS tags;
