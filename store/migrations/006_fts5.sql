-- +goose Up
-- Full-text search index on entries.
-- +goose StatementBegin
CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    title,
    summary,
    content,
    content='entries',
    content_rowid='id'
);
-- +goose StatementEnd

-- Populate FTS from existing entries.
INSERT INTO entries_fts(rowid, title, summary, content)
    SELECT id, title, summary, content FROM entries;

-- Keep FTS in sync with INSERT/UPDATE/DELETE on entries.
-- +goose StatementBegin
CREATE TRIGGER entries_fts_insert AFTER INSERT ON entries BEGIN
    INSERT INTO entries_fts(rowid, title, summary, content)
        VALUES (new.id, new.title, new.summary, new.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER entries_fts_delete AFTER DELETE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, title, summary, content)
        VALUES ('delete', old.id, old.title, old.summary, old.content);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER entries_fts_update AFTER UPDATE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, title, summary, content)
        VALUES ('delete', old.id, old.title, old.summary, old.content);
    INSERT INTO entries_fts(rowid, title, summary, content)
        VALUES (new.id, new.title, new.summary, new.content);
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS entries_fts_update;
DROP TRIGGER IF EXISTS entries_fts_delete;
DROP TRIGGER IF EXISTS entries_fts_insert;
DROP TABLE IF EXISTS entries_fts;
