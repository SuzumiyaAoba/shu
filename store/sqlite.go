package store

import (
	"database/sql"
	"fmt"
	"time"
)

// SQLiteOptions controls runtime settings for [SQLiteStore].
type SQLiteOptions struct {
	MaxOpenConns int
	BusyTimeout  time.Duration
}

// SQLiteStore implements [Store] using a SQLite database via the pure-Go
// modernc.org/sqlite driver (no CGo required).
//
// On initialization it enables WAL journal mode for better read concurrency,
// turns on foreign key enforcement (required for ON DELETE CASCADE), and
// applies the embedded schema migrations. The underlying [sql.DB] is safe for
// concurrent use; the store keeps max open connections at 1 to avoid SQLite
// lock contention while still allowing concurrent callers through the pool.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at the given DSN and
// returns a ready-to-use [SQLiteStore].
//
// The DSN is typically a file path (e.g. "/home/user/.shu/shu.db") or the
// special value ":memory:" for an in-memory database used in tests.
//
// Initialization steps:
//  1. Open the database connection.
//  2. Enable WAL journal mode for improved concurrent read performance.
//  3. Enable foreign key constraint enforcement (SQLite disables it by default).
//  4. Run all pending schema migrations tracked by the schema_migrations table.
//
// If any step fails, the database is closed and an error is returned.
func NewSQLiteStore(dsn string) (*SQLiteStore, error) {
	return NewSQLiteStoreWithOptions(dsn, nil)
}

// NewSQLiteStoreWithOptions opens (or creates) a SQLite database using the
// provided options.
func NewSQLiteStoreWithOptions(dsn string, options *SQLiteOptions) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	maxOpenConns := 1
	busyTimeout := 5 * time.Second
	if options != nil {
		if options.MaxOpenConns > 0 {
			maxOpenConns = options.MaxOpenConns
		}
		if options.BusyTimeout > 0 {
			busyTimeout = options.BusyTimeout
		}
	}

	// SQLite supports limited concurrency. Use a single connection to avoid
	// "database is locked" errors, especially with in-memory databases.
	db.SetMaxOpenConns(maxOpenConns)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set journal mode: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout/time.Millisecond)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.runMigrations(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the underlying database connection pool.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
