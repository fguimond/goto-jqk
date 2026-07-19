# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build    # build ./bin/goto-jqk with version metadata injected via -ldflags
make run      # go run ./cmd/goto-jqk run
make test     # go test ./...
make lint     # golangci-lint run
make tidy     # go mod tidy

go test ./internal/service -run TestGameService_CreateAndDelete   # single test
go run ./cmd/goto-jqk run --addr :9090                            # custom listen addr
```

Requires Go 1.25+. The server listens on `:8080` by default and serves interactive docs at `/docs`, plus `/openapi.json` and `/openapi.yaml`.

## Architecture

A layered REST API with dependencies pointing strictly inward: `handler → service → store`. The layers are composed in one place — `api.NewHandler` (`internal/api/server.go`) constructs `memory.NewGameStore()` → `service.NewGameService()` → `handler.NewGameHandler()` and registers operations. To add a resource, mirror this wiring there.

Key conventions that span multiple files:

- **Interfaces are defined at the point of use, not next to implementations.** The `GameStore` interface lives in `internal/service/game.go` (the consumer), while `internal/store/memory/` holds the concrete implementation. The `internal/store` package intentionally exports only concrete types and sentinel errors (e.g. `store.ErrNotFound`) — never storage interfaces. When adding storage behavior, extend the interface in the service package, not the store package.

- **HTTP layer is huma-driven.** Handlers register `huma.Operation`s (see `GameHandler.Register`) and use typed input/output structs whose struct tags (`json`, `path`, `format:"uuid"`, `minLength`, `doc`, `example`) generate the OpenAPI 3.1 schema and request validation automatically. Return errors via `huma.Error4xx/5xx` helpers; map sentinel store errors to HTTP status in the handler (e.g. `store.ErrNotFound` → `huma.Error404NotFound`). The API model struct (`handler.Game`) is deliberately separate from the domain model (`model.Game`).

- **CLI is cobra + viper, kubectl-style** (root command + subcommands in `internal/cli/`). Configuration precedence is flags > env vars (`GOTO_JQK_` prefix, `-` replaced with `_`) > `$HOME/.goto-jqk.yaml`. New flags must be bound with `viper.BindPFlag` to participate in this resolution; read them back via `viper.GetString(...)`, not the cobra flag directly.

- **Version metadata** (`internal/version`) is injected at link time by the Makefile's `-ldflags -X` and surfaced by `goto-jqk version` and the OpenAPI info block. `go run` builds show `dev`/`none`.

- **Logging** uses stdlib `log/slog` (JSON). `internal/api/middleware.go` wraps the mux with request logging; the logger is created in `cli/run.go` and passed down.

## Releases

Uses [release-please](https://github.com/googleapis/release-please) with Conventional Commits (`feat:`, `fix:`, `feat!:`). Merging the release PR tags the release and builds cross-platform binaries. Commit messages must follow Conventional Commits for versioning to work.
