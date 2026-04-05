# 011: Security Hardening

## Overview

コードベースの監査で発見されたセキュリティリスクを修正する。SQL インジェクション経路、
SSRF リスク、コマンドインジェクション、ファイルパーミッション問題を対象とする。

---

## Proposal 1: `buildEntriesColumnUpdate` のカラム名ホワイトリスト

### 現状の問題

`store/sqlite_entry_state.go:19` の `buildEntriesColumnUpdate` は `column` パラメータを
SQL 文字列に直接埋め込んでいる:

```go
return fmt.Sprintf(`UPDATE entries SET %s = NULL WHERE id IN (%s)`, column, placeholders), args
```

現在この関数の呼び出し元はすべて内部コード（`"read_at"`, `"starred_at"` のみ）だが、
将来新しい呼び出し元が追加された場合にSQLインジェクション経路となりうる。

### 提案する変更

カラム名をホワイトリストで検証するガードを追加する:

```go
var validEntryStateColumns = map[string]bool{
    "read_at":    true,
    "starred_at": true,
}

func buildEntriesColumnUpdate(column string, value any, ids []int64) (string, []any, error) {
    if !validEntryStateColumns[column] {
        return "", nil, fmt.Errorf("invalid column for entry state update: %q", column)
    }
    // ... 既存ロジック
}
```

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `store/sqlite_entry_state.go` | ホワイトリスト検証を追加、戻り値に error を追加 |

### 工数: Tiny (30分)

---

## Proposal 2: Feed URL の入力バリデーション (SSRF 防止)

### 現状の問題

`core/feed.go:51` の `AddFeed` と `core/discover.go:30` の `DiscoverFeeds` は
任意の URL をそのまま HTTP リクエストする。プライベート IP 範囲
(127.0.0.1, 10.x.x.x, 192.168.x.x, 169.254.x.x) や危険なスキーム
(`file://`, `javascript:`) に対するバリデーションがない。

CLI ツールとしてはリスクは低いが、将来的に Web サービスとして公開された場合に
SSRF 脆弱性となる。

### 提案する変更

`core/` に URL バリデーション関数を追加し、`AddFeed`・`DiscoverFeeds` の入口で呼ぶ:

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

### 注意点

- `--allow-private` フラグで開発用途のローカルフィード（localhost 上の RSS）を許可できるようにする
- 既存のフィードで localhost を使っているケースを壊さないよう、バリデーションは `AddFeed` のみに適用し、`FetchFeed` には適用しない

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/url_validate.go` | 新規: URL バリデーション関数 |
| `core/url_validate_test.go` | 新規: テスト |
| `core/feed.go` | `AddFeed` の先頭で `validateFeedURL` を呼ぶ |
| `core/discover.go` | `DiscoverFeeds` の先頭で `validateFeedURL` を呼ぶ |

### 工数: Small (1–2時間)

---

## Proposal 3: `openBrowser` のコマンドインジェクション防止

### 現状の問題

`cmd/entry_commands.go:255-266` の `openBrowser` は URL を `exec.Command` に渡している:

```go
func openBrowser(url string) error {
    switch runtime.GOOS {
    case "darwin":
        return exec.Command("open", url).Start()
    // ...
    }
}
```

URL はデータベースから取得されたものなので直接的なユーザー入力ではないが、
フィードに悪意のある URL が含まれていた場合、シェルメタ文字が解釈される可能性がある。

`exec.Command` は直接的なシェル呼び出しではないため実際のリスクは低い
（Go の `exec.Command` は argv 渡しでシェル経由しない）。ただし Windows の
`cmd /c start` パスはシェル経由であり、URL 中の `&` 等が解釈される。

### 提案する変更

Windows パスのみ修正が必要:

```go
case "windows":
    return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
```

また、URL が `http://` または `https://` で始まることを確認するガードを追加:

```go
func openBrowser(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
        return fmt.Errorf("refusing to open non-HTTP URL: %s", rawURL)
    }
    // ...
}
```

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `cmd/entry_commands.go` | `openBrowser` にスキーム検証を追加、Windows パスを `rundll32` に変更 |

### 工数: Tiny (30分)

---

## Proposal 4: データベースディレクトリのパーミッション修正

### 現状の問題

`app/app.go:102` でデータベースディレクトリを `0o755` (全ユーザー読み取り可能) で作成:

```go
if err := os.MkdirAll(dir, 0o755); err != nil {
```

RSS リーダーのデータベースには購読情報が含まれるため、他ユーザーから読み取れる
べきではない。

### 提案する変更

```go
if err := os.MkdirAll(dir, 0o700); err != nil {
```

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `app/app.go` | ディレクトリパーミッションを `0o755` → `0o700` に変更 |

### 工数: Tiny (5分)

---

## Priority Matrix

| Proposal | リスク | 工数 | 推奨優先度 |
|----------|--------|------|-----------|
| 1. カラムホワイトリスト | Low (内部コードのみ) | Tiny | **High** — 防御的プログラミング |
| 4. DB ディレクトリ権限 | Low | Tiny | **High** — 1行変更 |
| 3. openBrowser スキーム検証 | Low | Tiny | **High** — Windows パス修正 |
| 2. URL バリデーション | Medium (将来のSSRF) | Small | Medium — CLI ではリスク低 |

## 推奨実行順序

1. Proposal 4 — DB ディレクトリ権限 (5分)
2. Proposal 1 — カラムホワイトリスト (30分)
3. Proposal 3 — openBrowser スキーム検証 (30分)
4. Proposal 2 — URL バリデーション (1–2時間)

## 完了条件

- [ ] 全既存テストがパス
- [ ] 新規テスト追加 (特に Proposal 2)
- [ ] `golangci-lint run` クリーン
