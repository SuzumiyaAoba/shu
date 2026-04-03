package store

import (
	"embed"
	"fmt"
	"sort"
)

// migrationsFS embeds all SQL migration files from the migrations directory.
// Files are named with a numeric prefix (e.g. 001_, 002_) and executed in
// lexicographic order. Each migration runs at most once, tracked by the
// schema_migrations table.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations applies all embedded SQL migration files that have not yet been
// executed. It uses a schema_migrations table to track which files have already
// been applied, ensuring each migration runs exactly once and enabling safe
// ALTER TABLE statements that would fail if re-executed.
func (s *SQLiteStore) runMigrations() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		name := entry.Name()

		var count int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = ?`, name).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		data, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for migration %s: %w", name, err)
		}
		if _, err := tx.Exec(string(data)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}
