# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development

This project uses Nix for reproducible development environments.

```bash
nix develop                       # Enter dev shell (Go, gopls, golangci-lint, sqlite)
go build -o shu .                 # Build binary
go test ./...                     # Run all tests
go test ./store/ -v               # Run tests for a specific package
go test ./core/ -run TestFetchAll # Run a single test
golangci-lint run                 # Lint
```

Makefile shortcuts: `make build`, `make test`, `make lint`, `make clean`

## Architecture

Three-layer architecture with dependency injection, designed so `core/` can run outside the CLI (e.g., AWS Lambda):

- **`cmd/`** — CLI layer (Cobra). Composition root that wires `core.Service` with `store.SQLiteStore`. Global flags (`--db`, `--log-level`) and DB/Service initialization happen in `PersistentPreRunE` of `cmd/root.go`.
- **`core/`** — Business logic. Defines its own `Store` interface (`core/core.go`) and depends only on that interface, never on SQLite or CLI. `Service` struct holds the store, logger, and HTTP client.
- **`store/`** — SQLite storage. Implements `core.Store`. Schema migrations are embedded via `go:embed` and auto-applied on `NewSQLiteStore()`. Deduplication uses `INSERT OR IGNORE` with a `UNIQUE(feed_id, guid)` constraint.

Key constraint: `core/` must never import `cmd/` or `store/`. `store/` imports `core/` only for model types.

## Testing Patterns

- All tests use in-memory SQLite (`:memory:`) — no test fixtures or external databases.
- Core tests use `httptest.NewServer` with mock RSS XML to avoid network calls.
- Test helpers: `newTestService()` in `core/feed_test.go`, `newTestStore()` in `store/sqlite_test.go`.

## Key Dependencies

- `modernc.org/sqlite` — Pure Go SQLite driver (no CGo, simplifies cross-compilation)
- `github.com/mmcdole/gofeed` — RSS/Atom/JSON Feed parser
- `github.com/spf13/cobra` — CLI framework
- `log/slog` — Structured logging (stdlib)
