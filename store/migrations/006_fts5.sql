-- Full-text search index on entries.
CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    title,
    summary,
    content,
    content='entries',
    content_rowid='id'
);

-- Populate FTS from existing entries.
INSERT INTO entries_fts(rowid, title, summary, content)
    SELECT id, title, summary, content FROM entries;

-- Keep FTS in sync with INSERT/UPDATE/DELETE on entries.
CREATE TRIGGER entries_fts_insert AFTER INSERT ON entries BEGIN
    INSERT INTO entries_fts(rowid, title, summary, content)
        VALUES (new.id, new.title, new.summary, new.content);
END;

CREATE TRIGGER entries_fts_delete AFTER DELETE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, title, summary, content)
        VALUES ('delete', old.id, old.title, old.summary, old.content);
END;

CREATE TRIGGER entries_fts_update AFTER UPDATE ON entries BEGIN
    INSERT INTO entries_fts(entries_fts, rowid, title, summary, content)
        VALUES ('delete', old.id, old.title, old.summary, old.content);
    INSERT INTO entries_fts(rowid, title, summary, content)
        VALUES (new.id, new.title, new.summary, new.content);
END;
