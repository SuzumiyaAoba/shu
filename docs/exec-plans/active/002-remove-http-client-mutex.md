# 002: Remove Unnecessary HTTP Client Mutex

## Overview

The `Service` HTTP client is set only during construction and then remains read-only.
The protective `sync.RWMutex` is unnecessary and misleads readers into thinking the client is mutable.
Remove it to make immutability explicit.

## Current Problem

```go
// core/core.go
type Service struct {
    store    Store
    logger   *slog.Logger
    client   *http.Client
    clientMu sync.RWMutex  // ← unnecessary
}

func (s *Service) setHTTPClient(c *http.Client) {
    s.clientMu.Lock()
    defer s.clientMu.Unlock()
    s.client = c
}

func (s *Service) httpClient() *http.Client {
    s.clientMu.RLock()
    defer s.clientMu.RUnlock()
    return s.client
}
```

`setHTTPClient` is called only within `Option` functions inside `New`.
There is no path where `setHTTPClient` is called after `New` returns.

## Improvement Plan

### Step 1: Remove Mutex and Accessor Methods

```go
type Service struct {
    store  Store
    logger *slog.Logger
    client *http.Client
}
```

### Step 2: Option Functions Assign Directly

```go
func WithHTTPClient(c *http.Client) Option {
    return func(s *Service) {
        if c != nil {
            s.client = c
        }
    }
}

func WithHTTPClientWithUserAgent(c *http.Client) Option {
    return func(s *Service) {
        if c != nil {
            s.client = httpClientWithUserAgent(c)
        }
    }
}
```

### Step 3: Replace `s.httpClient()` with `s.client`

In `fetch_download.go`, change `s.httpClient().Do(req)` to `s.client.Do(req)`.

## Target Files

| File | Change |
|------|--------|
| `core/core.go` | Remove `clientMu`, `setHTTPClient`, `httpClient` methods; Option functions assign directly |
| `core/fetch_download.go` | `s.httpClient()` → `s.client` |

## Risks

- None. Code inspection confirms no path modifies client after construction
- `sync` package import may become unnecessary (if unused elsewhere)

## Completion Criteria

- [ ] `sync.RWMutex` removed from `Service`
- [ ] `setHTTPClient` and `httpClient` methods removed
- [ ] All existing tests pass
