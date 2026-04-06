# 011: Security Hardening

## Overview

Fix security risks discovered in the code audit. Address SQL injection pathways,
SSRF risks, command injection, and file permission issues.

---

## Proposal 1: Column Name Whitelist for `buildEntriesColumnUpdate`

### Current Issue

The `buildEntriesColumnUpdate` function at `store/sqlite_entry_state.go:19` embeds the `column` parameter
directly into the SQL string:

```go
return fmt.Sprintf(`UPDATE entries SET %s = NULL WHERE id IN (%s)`, column, placeholders), args
```

Currently all callers are internal code (only `"read_at"` and `"starred_at"`), but
this could become a SQL injection pathway if new callers are added in the future.

### Proposed Change

Add a guard that validates column names against a whitelist:

```go
var validEntryStateColumns = map[string]bool{
    "read_at":    true,
    "starred_at": true,
}

func buildEntriesColumnUpdate(column string, value any, ids []int64) (string, []any, error) {
    if !validEntryStateColumns[column] {
        return "", nil, fmt.Errorf("invalid column for entry state update: %q", column)
    }
    // ... existing logic
}
```

### Impact Scope

| File | Change |
|------|--------|
| `store/sqlite_entry_state.go` | Add whitelist validation, add error to return value |

### Effort: Tiny (30 minutes)

---

## Proposal 2: Feed URL Input Validation (Prevent SSRF)

### Current Issue

`AddFeed` at `core/feed.go:51` and `DiscoverFeeds` at `core/discover.go:30`
make HTTP requests with arbitrary URLs without validation for private IP ranges
(127.0.0.1, 10.x.x.x, 192.168.x.x, 169.254.x.x) or dangerous schemes
(`file://`, `javascript:`).

As a CLI tool, the risk is low, but this could become an SSRF vulnerability
if the service is published as a web service in the future.

### Proposed Change

Add a URL validation function to `core/` and call it at the entry of `AddFeed` and `DiscoverFeeds`:

```go
// core/url_validate.go
func validateFeedURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("unsupported scheme: %s (only http and https are allowed)", u.Scheme)
    }
    if u.Host == "" {
        return fmt.Errorf("URL must have a host")
    }
    host := u.Hostname()
    if isPrivateHost(host) {
        return fmt.Errorf("URL points to a private/loopback address: %s", host)
    }
    return nil
}
```

### Notes

- Allow local feeds (RSS on localhost) for development purposes via `--allow-private` flag
- Apply validation only to `AddFeed`, not to `FetchFeed`, to avoid breaking existing feeds using localhost

### Impact Scope

| File | Change |
|------|--------|
| `core/url_validate.go` | New: URL validation function |
| `core/url_validate_test.go` | New: Tests |
| `core/feed.go` | Call `validateFeedURL` at start of `AddFeed` |
| `core/discover.go` | Call `validateFeedURL` at start of `DiscoverFeeds` |

### Effort: Small (1–2 hours)

---

## Proposal 3: Prevent Command Injection in `openBrowser`

### Current Issue

The `openBrowser` function at `cmd/entry_commands.go:255-266` passes the URL to `exec.Command`:

```go
func openBrowser(url string) error {
    switch runtime.GOOS {
    case "darwin":
        return exec.Command("open", url).Start()
    // ...
    }
}
```

While the URL comes from the database (not direct user input), shell metacharacters could be
interpreted if a feed contains a malicious URL.

The actual risk is low since `exec.Command` does not invoke a shell directly (Go's `exec.Command` uses argv passing).
However, the Windows path `cmd /c start` does invoke a shell, where characters like `&` in the URL could be interpreted.

### Proposed Change

Only the Windows path needs fixing:

```go
case "windows":
    return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
```

Also add a guard to confirm the URL starts with `http://` or `https://`:

```go
func openBrowser(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
        return fmt.Errorf("refusing to open non-HTTP URL: %s", rawURL)
    }
    // ...
}
```

### Impact Scope

| File | Change |
|------|--------|
| `cmd/entry_commands.go` | Add scheme validation to `openBrowser`, change Windows path to `rundll32` |

### Effort: Tiny (30 minutes)

---

## Proposal 4: Fix Database Directory Permissions

### Current Issue

The database directory is created with permissions `0o755` (world-readable) at `app/app.go:102`:

```go
if err := os.MkdirAll(dir, 0o755); err != nil {
```

Since the RSS reader database contains subscription information, it should not be readable by other users.

### Proposed Change

```go
if err := os.MkdirAll(dir, 0o700); err != nil {
```

### Impact Scope

| File | Change |
|------|--------|
| `app/app.go` | Change directory permissions from `0o755` to `0o700` |

### Effort: Tiny (5 minutes)

---

## Priority Matrix

| Proposal | Risk | Effort | Recommended Priority |
|----------|------|--------|----------------------|
| 1. Column whitelist | Low (internal code only) | Tiny | **High** — Defensive programming |
| 4. DB directory permissions | Low | Tiny | **High** — Single-line change |
| 3. openBrowser scheme validation | Low | Tiny | **High** — Fix Windows path |
| 2. URL validation | Medium (future SSRF) | Small | Medium — Low risk as CLI tool |

## Recommended Execution Order

1. Proposal 4 — DB directory permissions (5 minutes)
2. Proposal 1 — Column whitelist (30 minutes)
3. Proposal 3 — openBrowser scheme validation (30 minutes)
4. Proposal 2 — URL validation (1–2 hours)

## Completion Checklist

- [ ] All existing tests pass
- [ ] New tests added (especially Proposal 2)
- [ ] `golangci-lint run` clean
