-- +goose Up
ALTER TABLE feeds ADD COLUMN description TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN language TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE feeds ADD COLUMN feed_type TEXT NOT NULL DEFAULT '';

ALTER TABLE entries ADD COLUMN content TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN author TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE entries ADD COLUMN categories TEXT NOT NULL DEFAULT '[]';
ALTER TABLE entries ADD COLUMN updated_at TEXT;
ALTER TABLE entries ADD COLUMN enclosures TEXT NOT NULL DEFAULT '[]';

-- +goose Down
-- SQLite does not support DROP COLUMN on older versions; no-op rollback.
SELECT 1;
