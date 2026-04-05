# 012: Concurrency Robustness

## Overview

フィード並行取得のワーカーにおけるパニックリカバリ、マイグレーションの
context 伝搬、エラースライスの事前確保を改善し、並行処理の堅牢性を高める。

---

## Proposal 1: フェッチワーカーのパニックリカバリ

### 現状の問題

`core/fetch_batch.go:73-99` のワーカーゴルーチンに `recover()` がない。
もしワーカー内でパニックが発生すると、`wg.Done()` が呼ばれず `wg.Wait()` が
永遠にブロックする (デッドロック)。

現状のコードでは gofeed パーサや HTTP 処理内でパニックが起きる可能性は低いが、
外部ライブラリのバグやメモリ不足時のランタイムパニックに対する防御がない。

### 提案する変更

ワーカー関数内に `recover` を追加し、パニックを通常のエラーに変換する:

```go
worker := func() {
    defer wg.Done()
    defer func() {
        if r := recover(); r != nil {
            errMu.Lock()
            fetchErrs = append(fetchErrs, fmt.Errorf("worker panic: %v", r))
            errMu.Unlock()
        }
    }()
    // ... 既存のワーカーロジック
}
```

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/fetch_batch.go` | ワーカー関数に `recover()` を追加 |
| `core/fetch_batch_test.go` | パニックリカバリのテスト追加 |

### 工数: Tiny (30分)

---

## Proposal 2: マイグレーションの Context 伝搬

### 現状の問題

`store/sqlite_migrations.go:44` で `context.Background()` を使用している:

```go
if _, err := provider.Up(context.Background()); err != nil {
```

これにより、マイグレーション実行中にアプリケーションがシャットダウンしても
マイグレーションをキャンセルできない。

### 提案する変更

`runMigrations` にコンテキストを渡すシグネチャに変更する:

```go
func (s *SQLiteStore) runMigrations(ctx context.Context) error {
    // ...
    if _, err := provider.Up(ctx); err != nil {
```

呼び出し元の `NewSQLiteStore` / `NewSQLiteStoreWithOptions` にも context を追加する。

### 注意点

- `NewSQLiteStore` のシグネチャ変更は破壊的変更
- `cmd/root.go` の `PersistentPreRunE` から context を渡せるため、実質的な変更は小さい
- goose の `provider.Up` は context.Context を受け取るため、API 上の制約はない

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `store/sqlite.go` | `NewSQLiteStore` / `NewSQLiteStoreWithOptions` に `context.Context` パラメータ追加 |
| `store/sqlite_migrations.go` | `runMigrations(ctx)` に変更 |
| `store/sqlite_test.go` | テストヘルパー更新 |
| `app/app.go` | `StoreOpener` 型に context を追加 |
| `cmd/root.go` | context を渡すよう更新 |

### 工数: Small (1–2時間)

---

## Proposal 3: フェッチエラースライスの事前確保

### 現状の問題

`core/fetch_batch.go:70` で `fetchErrs` が nil スライスとして宣言され、
エラー発生時に `append` で逐次拡張される:

```go
var (
    fetchErrs  []error
)
```

多数のフィード (1000+) でエラーが多発した場合、スライスの再配置が頻繁に発生する。

### 提案する変更

```go
fetchErrs := make([]error, 0, min(len(feeds), 64))
```

上限 64 をキャップとして事前確保する。実際にエラーが少ない場合のメモリ浪費を抑えつつ、
中程度のエラー数に対するアロケーションを削減する。

### 影響範囲

| ファイル | 変更 |
|----------|------|
| `core/fetch_batch.go` | `fetchErrs` の宣言を事前確保に変更 |

### 工数: Tiny (10分)

---

## Priority Matrix

| Proposal | 影響 | 工数 | 推奨優先度 |
|----------|------|------|-----------|
| 1. パニックリカバリ | High (デッドロック防止) | Tiny | **High** |
| 3. エラースライス事前確保 | Low (パフォーマンス) | Tiny | **High** — 1行変更 |
| 2. マイグレーション context | Medium (graceful shutdown) | Small | Medium — 破壊的変更あり |

## 推奨実行順序

1. Proposal 3 — エラースライス事前確保 (10分)
2. Proposal 1 — パニックリカバリ (30分)
3. Proposal 2 — マイグレーション context 伝搬 (1–2時間)

## 完了条件

- [ ] 全既存テストがパス
- [ ] Proposal 1: パニック発生時にデッドロックしないことを確認するテスト追加
- [ ] Proposal 2: context キャンセル時にマイグレーションが中断されることを確認
- [ ] `go vet ./...` および `golangci-lint run` クリーン
