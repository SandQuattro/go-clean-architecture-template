# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go 1.26 Clean Architecture template: REST API built on Fiber v2 + Huma v2 (OpenAPI 3.1) over PostgreSQL (pgx v5), with slog logging, Prometheus metrics, and OpenTelemetry tracing. Module name: `clean-arch-template`.

Note: huma is pinned to v2.37.0 — it is the last version whose `humafiber` adapter targets Fiber v2 (v2.38+ requires Fiber v3, whose otelfiber/fiberprometheus ecosystem is not ready yet).

## Commands

### Build & Run
- `make run` — run locally (`go run ./cmd/template/main.go`); requires Postgres and env vars (`DB_PASSWORD` and `PG_POOL_MAX` are `env-required` in config)
- `DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make cleandc` — full local stack via docker-compose (app + Postgres + Jaeger + Prometheus + Grafana + integration tests), wiping the Postgres volume first; `make dc` keeps the volume
- `make build` — build with version injected via ldflags (`version.Version`)

### Tests
- `make tests` — unit tests (excludes `integration-test/`)
- Single test: `go test -count=1 -run 'TestUserRepository' ./internal/usecase/repository/ -v`
- `make test-coverage` — coverage with `-race`, opens HTML report
- Integration tests (`integration-test/`) run as the `integration-tests` docker-compose service against the live app via env `HOST`/`PORT`; skip with `SKIP_INTEGRATION_TESTS=1`. They cover health probes, user CRUD, and transfer error paths (a successful transfer is unreachable via public API — new users start with zero balance)
- Load test: `k6 run load-test/load_test.js` (app must be running)

### Lint
- `make lint` — runs `golangci-lint` v2 (auto-installed into `./bin`) with `.golangci.yml` (config format v2, includes gofumpt as formatter)
- `golangci-lint fmt ./...` — format; `gofumpt` is not installed standalone
- Linter skips test files; `io/ioutil` denied by depguard; use `any` instead of `interface{}`

### Code generation & migrations
- `go generate ./...` — regenerate gomock mocks; each `interfaces.go` has a `//go:generate mockgen` directive producing `mocks.go` in the same package (mockgen from `go.uber.org/mock`)
- `make new-migration name=<migration_name>` — create a goose SQL migration in `migrations/` (single file with `-- +goose Up` / `-- +goose Down` sections)
- `make migrations-status` — show migration status against the local docker-compose Postgres (host port 5434)
- Migrations are applied automatically at app startup by the goose provider (`internal/app/migrations.go`) under a pg advisory session-lock; a failed migration aborts startup. For multi-replica production deployments prefer a separate Job/initContainer

## Architecture

Dependency rule (inner layers know nothing about outer ones): `entity` ← `usecase` ← `handler` ← `app`.

- `internal/entity` — domain types (User, Order, Transfer) and **domain sentinel errors** (`errors.go`: `ErrUserNotFound`, `ErrInsufficientFunds`, …). Money is `int64` in minimal currency units (100 = 1$). No transport tags beyond `json`.
- `internal/usecase` — business logic and validation (pagination bounds, user-name rules, transfer invariants). Inputs are command structs (`commands.go`). Imports only entity.
- `internal/usecase/repository` — pgx implementation of `usecase.UserRepository`. Raw SQL with `pgx.CollectRows`/`RowToStructByName`. Declares its own consumer-side `Transactor` interface.
- `internal/handler/rest/v1` — Huma handlers with transport DTOs (`schemas.go`), entity↔DTO mapping (`converter.go`), domain→HTTP error mapping (`errors.go`, `MapError`).
- `internal/app` — composition root: `app.New(cfg)` connects DB, applies migrations, builds Fiber (with Read/Write/Idle timeouts) and middlewares, wires DI; `app.Run(ctx)` serves until the context is cancelled, then drains with `ShutdownTimeout` and closes the DB pool. Startup errors propagate to `main` — no silent zombie process.
- `cmd/template/main.go` — config load, slog setup, OTel init, `signal.NotifyContext`; tracer shuts down after the server drains.
- `config` — cleanenv: `CONFIG_PATH` env, or `config/config.toml` / `config.json` from CWD, falling back to the package source dir; env overrides files. `DB.DSN()` is the single connection-string source (pool + migrator). Durations (HTTP timeouts) are env-only.
- `pkg/database` — pgxpool wrapper: functional options (min/max pool, timeouts), connect retries, and the transactor.

### Key conventions

- **Consumer-side interfaces**: each layer declares the interface it consumes in its own `interfaces.go` (usecase declares `UserRepository`, handler declares `UserUseCase`, repository declares `Transactor`), generates mocks alongside via mockgen, and pins the implementation with a compile-time assertion.
- **Not-found and other domain failures are sentinel errors** from `internal/entity/errors.go`: repository returns `entity.ErrUserNotFound` (never `(nil, nil)`); handler maps via `errors.Is` in `MapError`. New domain errors must be added to the `MapError` switch. Unknown errors become a generic 500 — internal error text never reaches clients (it is logged instead).
- **Transactions**: repositories hold `tx.DBGetter` + `Transactor` (github.com/Thiht/transactor/pgx). Always acquire the connection via `r.db(ctx)`. Multi-statement operations wrap logic in `WithinTransaction` (see `TransferMoney`). `TransferMoney` locks both account rows in one query (`WHERE id = ANY($1) ORDER BY id FOR UPDATE`) — deterministic lock order prevents deadlocks; keep that pattern for any multi-row locking.
- **Write operations are single-statement**: `UPDATE … RETURNING` / `DELETE` + `RowsAffected` distinguish "updated" from "not found" atomically — no read-then-write.
- **Request validation split**: structural validation lives in Huma schema tags (`minimum`, `minLength` → HTTP 422 before the handler runs; unknown body fields are rejected); business validation lives in the usecase (→ 400/404/409 via `MapError`). `PUT /user/{id}` takes the ID from the path only.
- **Tracing**: handlers do `ctx, span := otel.Tracer(...).Start(ctx, ...)` and pass the new ctx down; otelfiber owns the server span, handler spans are internal.
- **Testing per layer**: usecase and handler tests use gomock / an in-memory fake repo; repository tests use pgxmock with a `fakeTransactor`; integration tests hit the running HTTP API.

### Runtime endpoints

App listens on `HTTP_PORT` (default 8000; docker-compose publishes it as host port 9000): `/docs` (OpenAPI UI), `/metrics` (Go metrics), `/fiber` (HTTP metrics), `/livez`, `/readyz` (readiness pings the DB pool — returns 503 when Postgres is down); dev-only: `/monitor`, `/debug/pprof/`. Jaeger UI in compose: http://localhost:16686.
