# shu

Go 製 RSS Aggregator CLI。RSS フィードを収集し SQLite に保存する。

## 開発環境

```bash
nix develop  # Nix devshell に入る
```

## ビルド・実行

```bash
go build -o shu .
./shu --help
```

## テスト

```bash
go test ./...
```

## Lint

```bash
golangci-lint run
```

## 技術スタック

- Go 1.22+
- SQLite ドライバ: `modernc.org/sqlite` (Pure Go, CGo 不要)
- CLI: `github.com/spf13/cobra`
- RSS パース: `github.com/mmcdole/gofeed`
- ログ: `log/slog` (stdlib)

## アーキテクチャ

- `core/` - ビジネスロジック。CLI やインフラに依存しないこと
- `store/` - ストレージ層。`Store` インターフェースを通じてのみ `core` から利用
- `cmd/` - CLI 層 (Cobra)。`core` と `store` を結合するコンポジションルート
- テストでは `:memory:` SQLite DB を使用
