# 014: Feed Management Enhancements

## Overview

フィード管理の実用性を向上させる機能追加。HTTP リダイレクト時の URL 自動更新、
エントリの日付範囲フィルタリング、フィード発見のフォールバック戦略を実装する。

---

## Proposal 1: HTTP 301 リダイレクト時のフィード URL 自動更新

### 現状の問題

フィード提供元が URL を変更して 301 (Moved Permanently) を返した場合、
`go-retryablehttp` はリダイレクト先を自動的にフォローする。しかし、データベース上の
フィード URL は古いままとなり、毎回リダイレクトが発生する。

一部のフィードサーバーはリダイレクトを恒久的に提供し続けるが、将来的に古い URL が
削除される可能性がある。

### 提案する変更

`fetch_download.go` でレスポンスの最終 URL を追跡し、301 リダイレクト時に
フィード URL を自動更新する:

```go
type fetchedFeedDocument struct {
    body       []byte
    headers    http.Header
    finalURL   string // リダイレクト後の最終URL
    redirected bool   // 301リダイレクトがあったか
}
```

`fetchBodyConditional` を修正して、レスポンスの `Request.URL` (リダイレクト後の
最終 URL) を返すようにする:

```go
func fetchBodyConditional(ctx context.Context, client *http.Client, url, etag, lastModified string) ([]byte, http.Header, string, error) {
    // ...
    finalURL := resp.Request.URL.String()
    return body, resp.Header, finalURL, nil
}
```

persist 時に URL が変わっていたらログ出力し、store の `UpdateFeed` で URL を更新:

```go
if document.redirected && document.finalURL != feed.URL {
    logger.Info("feed URL redirected, updating", "old_url", feed.URL, "new_url", document.finalURL)
    if err := p.store.UpdateFeed(ctx, feed.ID, FeedUpdate{URL: &document.finalURL}); err != nil {
        logger.Warn("failed to update redirected feed URL", "error", err)
    }
}
```

### 注意点

- 301 のみ処理する (302/307 は一時的リダイレクトのため無視)
- `CheckRedirect` でリダイレクトチェーンを確認し、301 のみを判定
- URL 変更はベストエフォート (失敗してもフェッチ自体は成功とする)
- 重複チェック: リダイレクト先の URL が既に別のフィードとして登録されていた場合はスキップ

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/fetch_download.go` | `fetchBodyConditional` の戻り値に最終 URL を追加、リダイレクト検出 |
| `core/fetch_persist.go` | リダイレクト検出時の URL 更新ロジック |
| `core/fetch_download_test.go` | 301 リダイレクト時の URL 更新テスト |

### 工数: Medium (2–3時間)

---

## Proposal 2: エントリの日付範囲フィルタリング

### 現状の問題

`core/model.go` の `EntryFilter` は `FeedID`, `UnreadOnly`, `StarredOnly`, `Tag` での
フィルタリングに対応しているが、日付範囲 (`--published-after`, `--published-before`)
によるフィルタリングがない。

ユーザーが「今週の記事だけ見たい」「先月のスター記事を確認したい」といった
時系列ベースの操作を行えない。

### 提案する変更

`EntryFilter` に日付範囲フィールドを追加:

```go
type EntryFilter struct {
    // ... 既存フィールド
    PublishedAfter  *time.Time `json:"published_after"`
    PublishedBefore *time.Time `json:"published_before"`
}
```

`store/sqlite_entries.go` の `newEntryFilterQuery` にフィルタ条件を追加:

```go
if filter.PublishedAfter != nil {
    query.add(`published_at >= ?`, filter.PublishedAfter.Format(time.RFC3339))
}
if filter.PublishedBefore != nil {
    query.add(`published_at < ?`, filter.PublishedBefore.Format(time.RFC3339))
}
```

CLI フラグを追加:

```go
entriesCmd.Flags().StringVar(&publishedAfter, "published-after", "", "filter entries published after this date (YYYY-MM-DD)")
entriesCmd.Flags().StringVar(&publishedBefore, "published-before", "", "filter entries published before this date (YYYY-MM-DD)")
```

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/model.go` | `EntryFilter` に `PublishedAfter`/`PublishedBefore` 追加 |
| `store/sqlite_entries.go` | `newEntryFilterQuery` に日付条件追加 |
| `cmd/entry_commands.go` | `entries` コマンドに `--published-after`/`--published-before` フラグ追加 |
| `store/sqlite_entries_test.go` | 日付範囲フィルタのテスト追加 |

### 工数: Small (1–2時間)

---

## Proposal 3: フィード発見のフォールバック戦略

### 現状の問題

`core/discover.go` の `DiscoverFeeds` は HTML ページの
`<link rel="alternate">` タグのみを解析する。多くのサイトではこのタグが存在するが、
一部のサイト (特に静的サイトジェネレータ) では適切なタグがない。

### 提案する変更

`<link rel="alternate">` で見つからなかった場合のフォールバック戦略を追加:

1. **既知のパスを探索**: 一般的なフィードパス (`/feed`, `/feed.xml`, `/rss`,
   `/rss.xml`, `/atom.xml`, `/index.xml`, `/feed/atom`, `/feed/rss`) を HEAD リクエストで確認
2. **JSON Feed の検出**: `application/feed+json` の `<link>` 検出 (既存) に加え、
   `/.well-known/feed` パスも確認

```go
var commonFeedPaths = []string{
    "/feed", "/feed.xml", "/rss", "/rss.xml",
    "/atom.xml", "/index.xml", "/feed/atom", "/feed/rss",
}

func (d *FeedDiscovery) discoverByCommonPaths(ctx context.Context, baseURL string) []string {
    var found []string
    for _, path := range commonFeedPaths {
        candidate := resolveURL(baseURL, path)
        if d.isFeedURL(ctx, candidate) {
            found = append(found, candidate)
        }
    }
    return found
}

func (d *FeedDiscovery) isFeedURL(ctx context.Context, url string) bool {
    req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
    if err != nil {
        return false
    }
    resp, err := d.client.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return false
    }
    ct := resp.Header.Get("Content-Type")
    return strings.Contains(ct, "xml") || strings.Contains(ct, "rss") ||
        strings.Contains(ct, "atom") || strings.Contains(ct, "feed+json")
}
```

### 注意点

- フォールバック探索は `<link>` 解析で結果がなかった場合のみ実行
- HEAD リクエストを使い、ボディのダウンロードを避ける
- Content-Type が明示されていないサーバーへの対応として、2xx レスポンスの場合に
  GET で先頭数バイトを読んで XML/JSON かどうかを判定するオプション追加も検討
- 並行リクエストで高速化 (既知パスが多いため、直列だと遅い)

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/discover.go` | `discoverByCommonPaths` メソッド追加、`DiscoverFeeds` からフォールバック呼び出し |
| `core/discover_test.go` | フォールバック発見のテスト追加 |

### 工数: Medium (2–3時間)

---

## Proposal 4: per-feed fetch interval の CLI 公開

### 現状の問題

`core/model.go:55` に `FetchIntervalSec` フィールドが既に存在し、
`fetch_batch.go:56-61` で per-feed インターバルのスキップロジックも実装済みだが、
CLI からこの値を設定する手段がない。

`core/model.go:166-169` の `FeedUpdate` には `Title` と `URL` しかなく、
`FetchIntervalSec` が含まれていない。

### 提案する変更

1. `FeedUpdate` に `FetchIntervalSec` を追加:

```go
type FeedUpdate struct {
    Title            *string `json:"title"`
    URL              *string `json:"url"`
    FetchIntervalSec *int    `json:"fetch_interval_sec"`
}
```

2. `store/sqlite_feed.go` の `UpdateFeed` に対応カラムを追加

3. `cmd/feed_commands.go` の `update` コマンドにフラグ追加:

```go
updateCmd.Flags().DurationVar(&fetchInterval, "fetch-interval", 0,
    "per-feed fetch interval (e.g. 1h, 30m); 0 uses global default")
```

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/model.go` | `FeedUpdate` に `FetchIntervalSec` 追加 |
| `store/sqlite_feed.go` | `UpdateFeed` で `FetchIntervalSec` カラム更新対応 |
| `cmd/feed_commands.go` | `update` コマンドに `--fetch-interval` フラグ追加 |
| `store/sqlite_feed_test.go` | `UpdateFeed` のテスト追加 |

### 工数: Small (1時間)

---

## Priority Matrix

| Proposal | 影響 | 工数 | 推奨優先度 |
|----------|------|------|-----------|
| 4. per-feed interval CLI 公開 | Medium (既存ロジック活用) | Small | **High** — 既存機能の UI 欠落 |
| 2. 日付範囲フィルタ | Medium (UX 向上) | Small | **High** |
| 1. 301 リダイレクト URL 更新 | Medium (運用効率) | Medium | Medium |
| 3. フィード発見フォールバック | Medium (発見率向上) | Medium | Low — nice to have |

## 推奨実行順序

### Phase 1: 既存機能の補完 (2–3時間)
1. Proposal 4 — per-feed interval CLI 公開
2. Proposal 2 — 日付範囲フィルタ

### Phase 2: フィード管理の改善 (4–6時間)
3. Proposal 1 — 301 リダイレクト URL 更新
4. Proposal 3 — フィード発見フォールバック

## 完了条件

- [ ] 全既存テストがパス
- [ ] 各 Proposal に対応するテスト追加
- [ ] `golangci-lint run` クリーン
- [ ] CLI ヘルプメッセージの整合性確認
