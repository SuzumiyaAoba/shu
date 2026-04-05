# 009: Improve Codebase with New Library Introductions

## Overview

After auditing the shu codebase, this plan identifies areas where introducing Go libraries
would meaningfully improve code quality, developer experience, and user experience.
Each proposal is independent and preserves the existing three-layer architecture
(`core/` depends only on its own `Store` interface).

---

## Proposal 1: Table Output — `charmbracelet/lipgloss` + `charmbracelet/table`

### Current Problem

`cmd/render.go` uses `text/tabwriter` for all table formatting.
No borders, colors, or terminal-width-aware wrapping. Long titles break column alignment.

### Proposed Library

- **`github.com/charmbracelet/lipgloss`** + **`github.com/charmbracelet/table`**
  - De facto standard for terminal styling in Go.
  - Declarative color, border, and padding definitions.
  - Automatic column width calculation and text wrapping.

Alternative: `github.com/olekukonez/tablewriter` — lighter but limited styling.

### Benefits

- Significantly improved readability for feed/entry/stats tables
- `--no-color` flag for CI/pipe usage
- Terminal-width-aware column truncation (long URLs → `...`)

### Affected Files

| File | Change |
|------|--------|
| `cmd/render.go` | Replace `tabwriter` → `charmbracelet/table` |
| `cmd/root.go` | Add `--no-color` global flag |
| `go.mod` | Add dependencies |

### Effort: Small (1-2 hours)

---

## Proposal 2: Test Assertions — `google/go-cmp`

### Current Problem

Tests use manual `if got != want` comparisons throughout (`store/sqlite_test.go` is typical).
Deep struct comparisons produce boilerplate code and unhelpful error messages.

### Proposed Library

- **`github.com/google/go-cmp/cmp`**
  - Google's official struct comparison library. Human-readable diff output.
  - `cmpopts.IgnoreFields` to exclude timestamps like `FetchedAt`.
  - Natural integration with stdlib `testing` package.

Testify is avoided: existing tests use only stdlib `testing`, and `go-cmp` fits the project's
minimal-dependency philosophy better.

### Benefits

- Dramatically reduces field-by-field comparison boilerplate
- Test failure output shows diffs, making it immediately clear which fields differ

### Affected Files

| File | Change |
|------|--------|
| `store/sqlite_test.go` | Gradual migration from `if got.X != want.X` → `cmp.Diff` |
| `store/sqlite_feed_test.go` | Same |
| `core/fetch_test.go` | Entry comparisons → `cmp.Diff` |
| `go.mod` | Add `google/go-cmp` (test dependency only) |

### Effort: Small (1-2 hours, incremental)

---

## Proposal 3: Configuration File Support — `spf13/viper`

### Current Problem

All configuration is via command-line flags. Specifying `--db`, `--log-level`,
`--sqlite-busy-timeout` every invocation is tedious. Daemon mode (`shu run`)
`--interval` cannot be persisted.

### Proposed Library

- **`github.com/spf13/viper`**
  - Same author as Cobra. Provides YAML/TOML/JSON config + env var + flag
    priority-based merging.
  - Direct Cobra integration via `viper.BindPFlags`.

### Benefits

- Persist settings in `~/.config/shu/config.yaml`:
  ```yaml
  db: /path/to/shu.db
  log-level: warn
  fetch:
    interval: 15m
    workers: 5
  sqlite:
    busy-timeout: 10s
  ```
- Environment variable overrides: `SHU_DB`, `SHU_LOG_LEVEL`
- Priority: flags > env vars > config file > defaults

### Affected Files

| File | Change |
|------|--------|
| `cmd/root.go` | Viper init, config file loading, `BindPFlags` |
| `cmd/root.go` | Add `--config` flag |
| `app/app.go` | Reflect config-file-derived values in `Config` |
| `go.mod` | Add `spf13/viper` |

### Effort: Medium (2-4 hours)

---

## Proposal 4: Structured Error Handling (stdlib only)

### Current Problem

Errors use string-based wrapping via `fmt.Errorf("fetch feed %s: %w", ...)`.
`core/errors.go` defines sentinel errors but lacks structured error types
that carry context (which feed ID failed, etc.).

### Proposal

No external library. Add structured error types to `core/errors.go`:

```go
type FeedError struct {
    FeedID  int64
    FeedURL string
    Op      string
    Err     error
}

func (e *FeedError) Error() string {
    return fmt.Sprintf("%s feed %d (%s): %v", e.Op, e.FeedID, e.FeedURL, e.Err)
}

func (e *FeedError) Unwrap() error { return e.Err }
```

### Benefits

- CLI layer can branch display logic on error type
- Structured log fields extractable from error context
- Clean branching via `errors.As`

### Affected Files

| File | Change |
|------|--------|
| `core/errors.go` | Add `FeedError`, `EntryError` types |
| `core/feed.go` | Gradual migration from `fmt.Errorf` → `FeedError` |
| `core/fetch_download.go` | Structured download errors |
| `cmd/feed_commands.go` | Error-type-based display control |

### Effort: Medium (2-3 hours)

---

## Proposal 5: Progress Bar — `vbauerster/mpb`

### Current Problem

`shu fetch` fetches all feeds concurrently but produces no output until completion.
The `FetchObserver` interface already exists and fires events (started/skipped/completed),
but the CLI layer does not consume them.

### Proposed Library

- **`github.com/vbauerster/mpb/v8`** (multi-progress bar)
  - Per-feed progress bar display
  - Lightweight — avoids pulling in the full bubbletea TUI framework

### Benefits

- Real-time progress during `shu fetch`:
  ```
  Fetching feeds ████████████████████░░░░░░░░░░ 15/23 (2 skipped, 1 error)
  ```
- Implement `FetchObserver` and wire it into `cmd/feed_commands.go`
- `--quiet` flag to suppress progress (pipe/CI compatibility)

### Affected Files

| File | Change |
|------|--------|
| `cmd/feed_commands.go` | Create `FetchObserver` impl, wire to fetch command |
| `cmd/root.go` | Add `--quiet` global flag |
| `go.mod` | Add `vbauerster/mpb` |

### Effort: Medium (3-4 hours)

---

## Proposal 6: HTML to Text Conversion — `k3a/html2text`

### Current Problem

`shu entries --format markdown` outputs `e.Content` (HTML) as-is.
Raw HTML tags are displayed in the terminal, making content unreadable.

### Proposed Library

- **`github.com/k3a/html2text`**
  - Converts HTML to plain text. Zero dependencies.
  - Suitable for converting RSS entry `Content` fields.

### Benefits

- `shu entries --format text` renders HTML content as readable plain text
- Foundation for a future `shu read <entry-id>` command

### Affected Files

| File | Change |
|------|--------|
| `cmd/entry_commands.go` | Add `--format text` option, HTML→text conversion |
| `go.mod` | Add `k3a/html2text` |

### Effort: Small (1-2 hours)

---

## Proposal 7: Retryable HTTP Client — `hashicorp/go-retryablehttp`

### Current Problem

The HTTP client in `core/fetch_download.go` has no retry mechanism.
Transient failures (503, timeouts) immediately record errors.
After 5 consecutive `RecordFeedError` calls, a feed is auto-disabled.

### Proposed Library

- **`github.com/hashicorp/go-retryablehttp`**
  - Automatic retry with exponential backoff
  - Provides `*http.Client` adapter: `retryablehttp.NewClient().StandardClient()`
  - Drop-in replacement for existing `http.Client`

### Benefits

- Automatic retry for transient 5xx errors and connection timeouts (default 3 retries)
- Prevents unnecessary feed disabling due to transient failures
- Apply globally by modifying `defaultHTTPClient()` in `core/core.go`

### Affected Files

| File | Change |
|------|--------|
| `core/core.go` | Integrate retryablehttp into `defaultHTTPClient()` |
| `core/fetch_download.go` | Improve logging for post-retry errors |
| `go.mod` | Add `hashicorp/go-retryablehttp` |

### Effort: Small (1-2 hours)

---

## Proposal 8: Database Migrations — `pressly/goose`

### Current Problem

`store/sqlite_migrations.go` implements a custom migration runner.
No rollback (down migration), version display, dry run, or migration generation.

### Proposed Library

- **`github.com/pressly/goose/v3`**
  - `go:embed` integration support
  - Up/Down migrations
  - `goose status` equivalent for viewing applied versions
  - Compatible with pure Go SQLite driver (modernc.org/sqlite)

### Benefits

- Migration rollback capability (safer schema changes)
- `shu migrate status` command to inspect schema version

### Affected Files

| File | Change |
|------|--------|
| `store/sqlite_migrations.go` | Replace custom runner → goose |
| `store/sqlite.go` | Update migration call in `NewSQLiteStore` |
| `store/migrations/` | Rename existing SQL files to goose format |
| `cmd/root.go` | Add `shu migrate` subcommand (optional) |
| `go.mod` | Add `pressly/goose/v3` |

### Caveats

- Requires migration from existing `schema_migrations` table to goose's `goose_db_version`
- First-run auto-conversion code needed to avoid breaking existing user databases

### Effort: Large (4-6 hours)

---

## Priority Matrix

| Proposal | Impact | Effort | Recommended Priority |
|----------|--------|--------|---------------------|
| 2. go-cmp (test assertions) | Medium | Small | **High** — zero risk, incremental adoption |
| 7. go-retryablehttp (retry) | High | Small | **High** — prevents false feed disabling |
| 6. html2text (text conversion) | Medium | Small | **High** — immediate UX improvement |
| 1. charmbracelet/table (tables) | Medium | Small | Medium |
| 5. mpb (progress bar) | Medium | Medium | Medium |
| 3. viper (config file) | High | Medium | Medium |
| 4. structured errors (stdlib) | Medium | Medium | Low — no external library, pure refactoring |
| 8. goose (migrations) | Medium | Large | Low — current runner works well enough |

## Prerequisites

- Each proposal is independently implementable. No inter-dependencies.
- Proposals 1 and 5 can share the Charm stack (lipgloss) if adopted together.
- Only proposal 7 adds an external library to `core/` package.
  All others are scoped to `cmd/` or `store/` layers.

## Completion Criteria

- [ ] Tests added for each introduced library
- [ ] All existing tests pass
- [ ] `go vet ./...` and `golangci-lint run` are clean
- [ ] CLAUDE.md Key Dependencies section updated
