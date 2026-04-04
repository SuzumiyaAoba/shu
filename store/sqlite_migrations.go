package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/pressly/goose/v3"
)

// migrationsFS embeds all SQL migration files from the migrations directory.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations applies all pending goose migrations.
// On first run against a database that used the legacy schema_migrations table,
// it bootstraps goose by marking those versions as already applied so existing
// data is not touched.
func (s *SQLiteStore) runMigrations() error {
	if err := s.bootstrapFromLegacyMigrations(); err != nil {
		return fmt.Errorf("bootstrap goose from legacy migrations: %w", err)
	}

	// goose expects migration files at the root of the FS.
	migrationsSubFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migrations sub-fs: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		s.db,
		migrationsSubFS,
		goose.WithVerbose(false),
	)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// bootstrapFromLegacyMigrations converts the old schema_migrations table
// (filename TEXT PRIMARY KEY) to goose's version tracking so that existing
// databases are not re-migrated from scratch.
//
// If the legacy table does not exist this is a no-op. If goose's table already
// exists the conversion has already run and is also skipped.
func (s *SQLiteStore) bootstrapFromLegacyMigrations() error {
	var legacyExists int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`,
	).Scan(&legacyExists); err != nil || legacyExists == 0 {
		return nil // no legacy table — fresh database or already converted
	}

	var gooseExists int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='goose_db_version'`,
	).Scan(&gooseExists); err != nil || gooseExists > 0 {
		return nil // already converted
	}

	// Read applied filenames from the legacy table.
	rows, err := s.db.Query(`SELECT filename FROM schema_migrations ORDER BY filename`)
	if err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var versions []int64
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		if v := parseMigrationVersion(name); v > 0 {
			versions = append(versions, v)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Create goose tracking table and mark the baseline (version 0) as applied.
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS goose_db_version (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp     TIMESTAMP DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create goose_db_version: %w", err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1)`,
	); err != nil {
		return err
	}
	for _, v := range versions {
		if _, err := s.db.Exec(
			`INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)`, v,
		); err != nil {
			return err
		}
	}

	// Drop the legacy table — goose is now in charge.
	if _, err := s.db.Exec(`DROP TABLE schema_migrations`); err != nil {
		return fmt.Errorf("drop schema_migrations: %w", err)
	}
	return nil
}

// parseMigrationVersion extracts the numeric version prefix from a migration
// filename, e.g. "003_atom_fields.sql" → 3.
func parseMigrationVersion(filename string) int64 {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	v, _ := strconv.ParseInt(parts[0], 10, 64)
	return v
}
