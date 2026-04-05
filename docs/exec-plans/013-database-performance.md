# 013: Database Performance Optimization

## Overview

SQLite のパフォーマンスに直結する設定とインデックスを最適化する。
WAL モード有効化、不足インデックスの追加、デフォルト busy timeout の設定を行う。

---

## Proposal 1: SQLite WAL モードの有効化

### 現状の問題

SQLite のデフォルトジャーナルモードは `delete` であり、書き込みと読み取りが
互いにブロックする。`shu fetch` は複数ワーカーが並行してエントリを追加するため、
`SQLITE_BUSY` エラーが発生しやすい。

### 提案する変更

データベース接続確立後に WAL モードを有効化する:

```go
// store/sqlite.go
func (s *SQLiteStore) configurePragmas() error {
    pragmas := []string{
        `PRAGMA journal_mode = WAL`,
        `PRAGMA synchronous = NORMAL`,
        `PRAGMA foreign_keys = ON`,
    }
    for _, p := range pragmas {
        if _, err := s.db.Exec(p); err != nil {
            return fmt.Errorf("exec %s: %w", p, err)
        }
    }
    return nil
}
```

### 注意点

- WAL モードは一度設定すれば永続する (DB ファイルレベル)
- `synchronous = NORMAL` は WAL モードでは安全で、`FULL` より高速
- `foreign_keys = ON` は既存コードでも設定済みか確認が必要

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `store/sqlite.go` | PRAGMA 設定関数の追加、`NewSQLiteStore` から呼び出し |
| `store/sqlite_test.go` | WAL モード有効化の確認テスト |

### 工数: Small (1時間)

---

## Proposal 2: エントリ状態フィルタリング用インデックスの追加

### 現状の問題

`entries` テーブルには `feed_id` と `published_at` のインデックスがあるが、
頻繁に使われるフィルタ条件のインデックスが不足している:

- `read_at IS NULL` (`--unread` フィルタ): インデックスなし
- `starred_at IS NOT NULL` (`--starred` フィルタ): インデックスなし

エントリ数が増えるとこれらのフィルタクエリのパフォーマンスが劣化する。

### 提案する変更

新しいマイグレーションファイルを追加:

```sql
-- +goose Up
-- Partial indexes for common entry state filters.
CREATE INDEX IF NOT EXISTS idx_entries_unread ON entries(fetched_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_entries_starred ON entries(starred_at DESC)
    WHERE starred_at IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_entries_starred;
DROP INDEX IF EXISTS idx_entries_unread;
```

### 注意点

- SQLite の部分インデックス (partial index) は条件付きクエリを大幅に高速化する
- インデックスサイズは未読/スターのエントリ数に比例し、全エントリのインデックスより小さい
- 既存データに対しても `CREATE INDEX IF NOT EXISTS` でマイグレーション時に自動構築される

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `store/migrations/009_entry_state_indexes.sql` | 新規: 部分インデックス定義 |

### 工数: Tiny (15分)

---

## Proposal 3: デフォルト busy timeout の設定

### 現状の問題

`--sqlite-busy-timeout` のデフォルト値が `0` (タイムアウトなし) である
(`cmd/root.go:90`)。これは `shu fetch` の並行ワーカーがデータベースロックの
競合で即座に `SQLITE_BUSY` エラーを返すことを意味する。

ユーザーが明示的に `--sqlite-busy-timeout 5s` を指定しない限り、この問題は発生しうる。

### 提案する変更

デフォルト値を `5s` に変更する:

```go
rootCmd.PersistentFlags().DurationVar(&sqliteBusyTimeout, "sqlite-busy-timeout", 5*time.Second, "SQLite busy timeout (e.g. 5s)")
```

また、`sqliteOptionsFromFlags` のゼロ値チェックを調整して、デフォルトが正しく
適用されるようにする。

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `cmd/root.go` | `--sqlite-busy-timeout` のデフォルト値を `5s` に変更 |
| `cmd/root.go` | `sqliteOptionsFromFlags` のゼロ値チェック調整 |

### 工数: Tiny (15分)

---

## Proposal 4: FTS5 検索クエリの最適化

### 現状の問題

`store/sqlite_entries.go:166-168` の FTS5 検索がサブクエリ + JOIN パターンを使用:

```sql
SELECT ... FROM entries WHERE id IN (SELECT rowid FROM entries_fts WHERE entries_fts MATCH ?)
ORDER BY fetched_at DESC LIMIT ? OFFSET ?
```

このパターンでは FTS5 の結果全体をサブクエリで生成した後にソートが行われる。
エントリ数が多い場合にパフォーマンスが劣化する可能性がある。

### 提案する変更

FTS5 のランキング機能を活用した代替クエリを検証し、パフォーマンスが向上する
場合のみ適用する:

```sql
SELECT ... FROM entries e
INNER JOIN entries_fts f ON e.id = f.rowid
WHERE f.entries_fts MATCH ?
ORDER BY e.fetched_at DESC
LIMIT ? OFFSET ?
```

### 注意点

- 実際のパフォーマンス差は `EXPLAIN QUERY PLAN` で検証する必要がある
- FTS5 の `rank` 列を使ったスコアベースのソートも検討に値する
- ベンチマークなしでの変更は避ける

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `store/sqlite_entries.go` | `SearchEntriesPage` のクエリ変更 (ベンチマーク結果次第) |
| `store/sqlite_entries_test.go` | 検索パフォーマンステスト追加 |

### 工数: Medium (2–3時間、ベンチマーク含む)

---

## Priority Matrix

| Proposal | 影響 | 工数 | 推奨優先度 |
|----------|------|------|-----------|
| 3. busy timeout デフォルト | High (SQLITE_BUSY 防止) | Tiny | **High** — 即効性 |
| 1. WAL モード | High (読み書き並行性) | Small | **High** |
| 2. 状態フィルタインデックス | Medium (クエリ高速化) | Tiny | **High** |
| 4. FTS5 クエリ最適化 | Medium | Medium | Low — ベンチマーク必要 |

## 推奨実行順序

### Phase 1: 即効性のある変更 (30分)
1. Proposal 3 — busy timeout デフォルト
2. Proposal 2 — 状態フィルタインデックス

### Phase 2: 構造的改善 (1時間)
3. Proposal 1 — WAL モード

### Phase 3: 検証必要 (2–3時間)
4. Proposal 4 — FTS5 最適化 (ベンチマーク結果次第で適用)

## 完了条件

- [ ] 全既存テストがパス
- [ ] WAL モードが有効になっていることを確認するテスト
- [ ] `EXPLAIN QUERY PLAN` でインデックスが使用されていることを確認
- [ ] `golangci-lint run` クリーン
