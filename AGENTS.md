# Repository Guidelines

## Project Structure & Module Organization
`shu` is a Go CLI RSS/Atom reader with a layered layout. `main.go` is the entrypoint. `cmd/` contains Cobra commands and output rendering. `app/` wires runtime dependencies such as store, logger, and service setup. `core/` holds business logic and fetch workflows; `core/fetch/` isolates feed download and persistence steps. `store/` implements SQLite-backed persistence, with schema files in `store/migrations/`. `model/` contains shared domain types. Design notes and execution plans live in `docs/exec-plans/`.

## Build, Test, and Development Commands
Use the Nix shell for a reproducible toolchain:

```bash
nix develop
make build      # build ./shu
make test       # run all Go tests
make lint       # run golangci-lint
make clean      # remove the local binary
```

Direct Go commands are also common inside the dev shell, for example `go test ./core/...` or `go test ./cmd -run TestFetchCmdAll`.

## Coding Style & Naming Conventions
Follow standard Go style and always format with `gofmt`/`go fmt`. Keep packages lowercase and focused by layer; `core/` must not import `cmd/` or `store/`. Exported names use `PascalCase`, internal helpers use `camelCase`, and tests live beside implementation files as `*_test.go`. Prefer table-driven tests for command and store behavior, and keep CLI output formats consistent across commands (`--json`, `--yaml`, table output).

## Testing Guidelines
Run `make test` before opening a PR. This repository relies on Go’s `testing` package, in-memory SQLite (`:memory:`), and `httptest` servers rather than external services. Name tests `TestXxx` and keep helpers local to the package when possible, such as `newTestStore()` and `newTestService()`. Add regression tests for bug fixes and package-level tests for new CLI flags or persistence logic.

## Commit & Pull Request Guidelines
Recent history favors short, imperative subjects with conventional prefixes such as `feat:`, `fix:`, `refactor:`, and `docs:`. Keep commits scoped to one concern. PRs should explain the user-visible change, note any schema or command-surface impact, link related issues or plans, and include example output when CLI formatting changes.

## Configuration & Data
The default database path is `~/.shu/shu.db`; use `--db` for isolated local testing. Prefer `--sqlite-busy-timeout` and `--sqlite-max-open-conns` for SQLite tuning instead of hardcoding environment-specific values.
