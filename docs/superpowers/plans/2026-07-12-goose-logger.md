# Goose Migrations + Logger Interface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Перевести миграции с golang-migrate на goose v3 (provider API, session lock) и ввести собственный ctx-first интерфейс логгера с реализациями slog и zerolog, выбираемыми через `LOG_BACKEND`.

**Architecture:** Часть 1 заменяет мигратор в `internal/app/migrations.go` на `goose.NewProvider` поверх `database/sql` (драйвер pgx stdlib), файлы миграций конвертируются в одно-файловый goose-формат. Часть 2 добавляет интерфейс `logger.Logger` (реализации — файлы внутри `pkg/logger`, иначе цикл импортов между интерфейсом, фабрикой и реализациями) и заменяет глобальный slog на DI по всем инфраструктурным точкам; usecase/repository не логируют и не меняются.

**Tech Stack:** Go 1.26, pressly/goose/v3 v3.27.2, jackc/pgx/v5 stdlib, rs/zerolog v1.35.1, log/slog, OpenTelemetry trace.

**Spec:** `docs/superpowers/specs/2026-07-12-goose-logger-design.md`

## Global Constraints

- goose закрепить на **v3.27.2**, zerolog на **v1.35.1**; после изменения зависимостей — `go mod tidy`.
- Формат и уровни логов не меняются: prod → JSON с ключом `message`, dev → text/console; уровень из `LOG_LEVEL`, `DEBUG=true` → Debug.
- Ошибка миграции или неизвестный `LOG_BACKEND` валит старт приложения.
- Хук post-edit линтит по одному файлу: ложные `undefined` для соседей по пакету игнорировать, проверять `go build ./...`; форматировать `golangci-lint fmt <paths>` (standalone gofumpt не установлен).
- Каждая часть заканчивается зелёным `go build ./... && go vet ./... && go test -count=1 $(go list ./... | grep -v integration-test) && golangci-lint run ./...` и полным стеком `DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make cleandc` (все интеграционные тесты PASS).
- Коммиты small и по шаблону репозитория; в конце сообщения строка `Claude-Session: https://claude.ai/code/session_01X9tT9pYdtbSBGZ9uSGxo4m`.

---

# Часть 1 — goose

### Task 1: Зависимость goose + конфиг MigrationsDir

**Files:**
- Modify: `go.mod` (через go get)
- Modify: `config/config.go` (DB struct)
- Test: `config/config_test.go`

**Interfaces:**
- Produces: `config.DB.MigrationsDir string` (env `MIGRATIONS_DIR`, default `migrations`) — используется Task 3.

- [ ] **Step 1: Написать падающий тест на дефолт MigrationsDir**

В `config/config_test.go` внутрь `TestLoadConfig` после существующих ассертов добавить:

```go
	assert.Equal(t, "migrations", cfg.DB.MigrationsDir)
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test -count=1 -run TestLoadConfig ./config/ -v`
Expected: FAIL — `cfg.DB.MigrationsDir undefined` (ошибка компиляции — это валидный red для нового поля).

- [ ] **Step 3: Добавить поле в config**

В `config/config.go`, struct `DB`, после поля `SSLMode`:

```go
		MigrationsDir     string `json:"migrations_dir" toml:"migrations_dir" env:"MIGRATIONS_DIR" env-default:"migrations"`
```

- [ ] **Step 4: Тест зелёный**

Run: `go test -count=1 -run TestLoadConfig ./config/ -v`
Expected: PASS

- [ ] **Step 5: Добавить зависимость goose**

```bash
go get github.com/pressly/goose/v3@v3.27.2 && go mod tidy
```
Expected: `go.mod` содержит `github.com/pressly/goose/v3 v3.27.2`.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum config/config.go config/config_test.go
git commit -m "feat: add goose dependency and MIGRATIONS_DIR config"
```

### Task 2: Конвертация миграций в goose-формат

**Files:**
- Create: `migrations/20230101000000_create_users_table.sql`
- Create: `migrations/20230101000001_create_orders_table.sql`
- Create: `migrations/20250209000001_create_transactions_table.sql`
- Create: `migrations/20260712000001_money_bigint_fk_indexes.sql`
- Delete: все 8 файлов `migrations/*.up.sql` и `migrations/*.down.sql`

**Interfaces:**
- Produces: каталог `migrations/` в goose-формате — читается Task 3 через `os.DirFS`.

- [ ] **Step 1: Создать 4 файла goose-формата**

`migrations/20230101000000_create_users_table.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS users
(
    id   BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    balance DECIMAL(15,2) NOT NULL CHECK (balance >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE users;
```

`migrations/20230101000001_create_orders_table.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS orders
(
    id   BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    amount BIGINT NOT NULL
);

-- +goose Down
DROP TABLE orders;
```

`migrations/20250209000001_create_transactions_table.sql`:
```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS transactions
(
    id           BIGSERIAL PRIMARY KEY,
    from_user_id BIGINT         NOT NULL,
    to_user_id   BIGINT         NOT NULL,
    amount       DECIMAL(15, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE transactions;
```

`migrations/20260712000001_money_bigint_fk_indexes.sql`:
```sql
-- +goose Up
-- Деньги переводятся в BIGINT (минимальные единицы валюты, 100 центов = 1$):
-- целочисленная арифметика без ошибок округления двоичной запятой.
ALTER TABLE users
    ALTER COLUMN balance TYPE BIGINT USING ROUND(balance * 100)::BIGINT,
    ALTER COLUMN balance SET DEFAULT 0;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;

-- Ссылочная целостность и индексы под FK-поиски.
ALTER TABLE orders
    ADD CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_from_user FOREIGN KEY (from_user_id) REFERENCES users (id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_transactions_to_user FOREIGN KEY (to_user_id) REFERENCES users (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_transactions_from_user_id ON transactions (from_user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_user_id ON transactions (to_user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_to_user_id;
DROP INDEX IF EXISTS idx_transactions_from_user_id;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS fk_transactions_to_user,
    DROP CONSTRAINT IF EXISTS fk_transactions_from_user;

DROP INDEX IF EXISTS idx_orders_user_id;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS fk_orders_user;

ALTER TABLE transactions
    ALTER COLUMN amount TYPE DECIMAL(15, 2) USING amount / 100.0;

ALTER TABLE users
    ALTER COLUMN balance DROP DEFAULT,
    ALTER COLUMN balance TYPE DECIMAL(15, 2) USING balance / 100.0;
```

- [ ] **Step 2: Удалить старые пары**

```bash
git rm migrations/*.up.sql migrations/*.down.sql
```
Expected: в `migrations/` остаются ровно 4 `.sql` файла.

- [ ] **Step 3: Commit**

```bash
git add migrations/
git commit -m "feat: convert migrations to goose single-file format"
```

### Task 3: migrations.go на goose provider

**Files:**
- Modify: `internal/app/migrations.go` (полная замена содержимого)
- Modify: `internal/app/app.go` (сигнатура `New` + вызов applyMigrations)
- Modify: `cmd/template/main.go` (blank-import, вызов app.New)

**Interfaces:**
- Consumes: `config.DB.DSN() string`, `config.DB.MigrationsDir` (Task 1).
- Produces: `app.New(ctx context.Context, cfg *config.Config) (*App, error)` — main передаёт ctx из `signal.NotifyContext`; внутренняя `applyMigrations(ctx context.Context, cfg config.DB) error`.

- [ ] **Step 1: Переписать `internal/app/migrations.go`**

```go
package app

import (
	"database/sql"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"clean-arch-template/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

const (
	defaultAttempts = 3
	defaultTimeout  = time.Second
)

// applyMigrations применяет миграции при старте через goose. Любая ошибка
// возвращается наверх — сервис не должен принимать трафик на битой схеме.
// Session-lock (pg advisory lock) защищает от параллельного применения
// несколькими репликами. В продакшене предпочтителен отдельный Job/initContainer.
func applyMigrations(ctx context.Context, cfg config.DB) error {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return fmt.Errorf("migrate: open db: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			slog.Error("migrate: close db", slog.String("error", cerr.Error()))
		}
	}()

	for attempts := defaultAttempts; attempts > 0; attempts-- {
		err = db.PingContext(ctx)
		if err == nil {
			break
		}
		slog.Debug(fmt.Sprintf("migrate: postgres is trying to connect, attempts left: %d", attempts))
		time.Sleep(defaultTimeout)
	}
	if err != nil {
		return fmt.Errorf("migrate: postgres connect: %w", err)
	}

	sessionLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("migrate: session locker: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS(cfg.MigrationsDir),
		goose.WithSessionLocker(sessionLocker),
	)
	if err != nil {
		return fmt.Errorf("migrate: provider: %w", err)
	}
	defer func() {
		if cerr := provider.Close(); cerr != nil {
			slog.Error("migrate: close provider", slog.String("error", cerr.Error()))
		}
	}()

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: up: %w", err)
	}

	if len(results) == 0 {
		slog.Info("Migrate: no change")
		return nil
	}

	slog.Info(fmt.Sprintf("Migrate: applied %d migrations", len(results)))

	return nil
}
```

- [ ] **Step 2: Обновить `internal/app/app.go`**

Сигнатура и вызов:
```go
func New(ctx context.Context, cfg *config.Config) (*App, error) {
```
и внутри:
```go
	if err := applyMigrations(ctx, cfg.DB); err != nil {
```
(остальное тело `New` не меняется).

- [ ] **Step 3: Обновить `cmd/template/main.go`**

Удалить строку `_ "github.com/golang-migrate/migrate/v4/database/postgres"` из импортов; вызов заменить на:
```go
	application, err := app.New(ctx, cfg)
```

- [ ] **Step 4: Убрать golang-migrate из зависимостей и собрать**

```bash
go mod tidy && go build ./... && go vet ./...
```
Expected: сборка зелёная; в `go.mod` больше нет `github.com/golang-migrate/migrate/v4`.

- [ ] **Step 5: Юнит-тесты**

Run: `go test -count=1 $(go list ./... | grep -v integration-test)`
Expected: все PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/migrations.go internal/app/app.go cmd/template/main.go go.mod go.sum
git commit -m "feat: switch startup migrations to goose provider with session lock"
```

### Task 4: Makefile, docs, полный стек

**Files:**
- Modify: `Makefile` (секция MIGRATIONS)
- Modify: `README.md` (Technologies, TODO, baseline-заметка)
- Modify: `CLAUDE.md` (команды и заметка про goose)

**Interfaces:**
- Consumes: каталог `migrations/` (Task 2).

- [ ] **Step 1: Заменить секцию MIGRATIONS в Makefile**

Удалить блок `.install-migrate`/`MIGRATE_VERSION` целиком, вместо него:

```make
# ---------------------------------- MIGRATIONS ---------------------------------
GOOSE_VERSION = v3.27.2
GOOSE = $(PROJECT_BIN)/goose

DB_USER ?= postgres
DB_PASSWORD ?= admin
DB_HOST ?= localhost
DB_PORT ?= 5434
DB_NAME ?= demo

.PHONY: .install-goose
.install-goose:
	[ -f $(GOOSE) ] || GOBIN=$(PROJECT_BIN) go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION)

.PHONY: new-migration
new-migration: .install-goose ## create new sql migration: make new-migration name=add_something
	$(GOOSE) -dir ./migrations create $(name) sql

.PHONY: migrations-status
migrations-status: .install-goose ## show migrations status against local db
	$(GOOSE) -dir ./migrations postgres "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable" status
```

- [ ] **Step 2: Проверить генерацию миграции**

Run: `make new-migration name=smoke_check && ls migrations/ && rm migrations/*smoke_check.sql`
Expected: создаётся `migrations/<timestamp>_smoke_check.sql` с секциями `-- +goose Up/Down`; файл удаляем.

- [ ] **Step 3: README и CLAUDE.md**

README: в Technologies заменить строку про golang-migrate на
`- Migrations: goose (pressly/goose v3), применяются автоматически при старте под pg advisory lock; ошибка миграции валит старт`;
в TODO отметить `- [x] Перевести миграции на goose`; добавить подраздел:

```markdown
## Baseline существующей БД (переход с golang-migrate)
Шаблон предполагает свежую БД. Если схема уже создана golang-migrate:
1. Убедитесь, что схема соответствует последней миграции.
2. Пометьте миграции применёнными без выполнения: `./bin/goose -dir ./migrations postgres "<DSN>" up --no-versioning` НЕ выполнять; вместо этого зафиксируйте версии: `./bin/goose -dir ./migrations postgres "<DSN>" fix` не требуется — просто выполните для каждой версии `INSERT INTO goose_db_version (version_id, is_applied) VALUES (<version>, true);`
3. Таблица `schema_migrations` от golang-migrate больше не используется, её можно удалить.
```

CLAUDE.md: в разделе «Code generation & migrations» заменить упоминания golang-migrate:
`make new-migration name=<name>` → goose-файл с `-- +goose Up/Down`; `make migrations-status`; миграции применяет goose provider под session-lock, ошибка валит старт.

- [ ] **Step 4: Полный стек**

Run: `DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make cleandc` (в фоне, следить за логами)
Expected: `Migrate: applied 4 migrations`, `Starting server on port: 8000`, все интеграционные тесты PASS, `tests exited with code 0`. Проверить таблицу: `docker exec postgres psql -U postgres -d demo -c "SELECT version_id FROM goose_db_version ORDER BY version_id;"` → 0 и 4 версии (goose пишет строку 0 + по строке на миграцию).

- [ ] **Step 5: Commit**

```bash
git add Makefile README.md CLAUDE.md
git commit -m "feat: goose CLI targets and migration docs"
```

---

# Часть 2 — интерфейс логгера

### Task 5: Интерфейс, otel-атрибуты, slog-реализация

**Files:**
- Create: `pkg/logger/logger.go` (интерфейс + фабрика)
- Create: `pkg/logger/otel.go`
- Create: `pkg/logger/slogx.go`
- Delete: старое содержимое `pkg/logger/logger.go` (SetupLogger)
- Test: `pkg/logger/logger_test.go`

**Interfaces:**
- Consumes: `config.Config` (App.Environment, App.Debug, Log.Level; поле `Log.Backend` добавляется в Task 7).
- Produces:
  - `type Logger interface { Debug(ctx context.Context, msg string, args ...any); Info(...); Warn(...); Error(...); With(args ...any) Logger }`
  - `func newSlogLogger(cfg *config.Config, out io.Writer) *slogLogger`
  - `func traceArgs(ctx context.Context) []any` — `["trace_id", <hex>, "span_id", <hex>]` или nil.

- [ ] **Step 1: Написать падающие тесты**

`pkg/logger/logger_test.go`:
```go
package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"clean-arch-template/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func prodConfig() *config.Config {
	cfg := &config.Config{}
	cfg.App.Environment = "prod"
	return cfg
}

func ctxWithSpan(t *testing.T) context.Context {
	t.Helper()

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x02},
		TraceFlags: trace.FlagsSampled,
	})
	require.True(t, spanCtx.IsValid())

	return trace.ContextWithSpanContext(context.Background(), spanCtx)
}

func lastJSONLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))

	return entry
}

func TestSlogLoggerWritesJSONInProd(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf)

	log.Info(context.Background(), "hello", "user_id", 42)

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "hello", entry["message"])
	assert.Equal(t, float64(42), entry["user_id"])
}

func TestSlogLoggerLevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf) // prod default: Info

	log.Debug(context.Background(), "invisible")

	assert.Empty(t, buf.Bytes())
}

func TestSlogLoggerWith(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf).With("component", "test")

	log.Info(context.Background(), "hello")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "test", entry["component"])
}

func TestSlogLoggerTraceCorrelation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf)

	log.Info(ctxWithSpan(t), "traced")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "01000000000000000000000000000000", entry["trace_id"])
	assert.Equal(t, "0200000000000000", entry["span_id"])
}

func TestTraceArgsWithoutSpan(t *testing.T) {
	t.Parallel()

	assert.Nil(t, traceArgs(context.Background()))
}
```

- [ ] **Step 2: Убедиться, что тесты падают**

Run: `go test -count=1 ./pkg/logger/ -v`
Expected: FAIL — `undefined: newSlogLogger`, `undefined: traceArgs`.

- [ ] **Step 3: Реализация**

`pkg/logger/logger.go` (полная замена файла, SetupLogger удаляется — потребители переводятся в Task 8):
```go
// Package logger — общий интерфейс логгера; реализации (slog, zerolog)
// живут в этом же пакете: вынос в подпакеты создал бы цикл импортов
// интерфейс ↔ фабрика ↔ реализация (метод With возвращает Logger).
package logger

import "context"

type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
	// With возвращает логгер с добавленными атрибутами (args — пары key/value).
	With(args ...any) Logger
}
```

`pkg/logger/otel.go`:
```go
package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// traceArgs достаёт из ctx активный спан; при валидном спане возвращает
// пары атрибутов для корреляции логов с трейсами.
func traceArgs(ctx context.Context) []any {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	return []any{
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	}
}
```

`pkg/logger/slogx.go`:
```go
package logger

import (
	"context"
	"io"
	"log/slog"

	"clean-arch-template/config"
)

type slogLogger struct {
	l *slog.Logger
}

var _ Logger = (*slogLogger)(nil)

// newSlogLogger — реализация на stdlib slog: prod → JSON с ключом message,
// иначе text с source-позициями; уровень из LOG_LEVEL, DEBUG=true → Debug.
func newSlogLogger(cfg *config.Config, out io.Writer) *slogLogger {
	level := cfg.Level
	if cfg.Debug {
		level = slog.LevelDebug
	}

	var handler slog.Handler

	if cfg.Environment == "prod" {
		renameMsgKey := func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		}
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: renameMsgKey,
		})
	} else {
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	}

	return &slogLogger{l: slog.New(handler)}
}

func (s *slogLogger) Debug(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelDebug, msg, args)
}

func (s *slogLogger) Info(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelInfo, msg, args)
}

func (s *slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelWarn, msg, args)
}

func (s *slogLogger) Error(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelError, msg, args)
}

func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.l.With(args...)}
}

func (s *slogLogger) log(ctx context.Context, level slog.Level, msg string, args []any) {
	if tr := traceArgs(ctx); tr != nil {
		args = append(append(make([]any, 0, len(args)+len(tr)), args...), tr...)
	}
	s.l.Log(ctx, level, msg, args...)
}
```

- [ ] **Step 4: Тесты зелёные**

Run: `go test -count=1 ./pkg/logger/ -v`
Expected: PASS (5 тестов). Примечание: `go build ./...` на этом шаге СЛОМАН (потребители SetupLogger) — это ожидаемо до Task 8; коммит только пакета logger.

- [ ] **Step 5: Commit**

```bash
git add pkg/logger/
git commit -m "feat: logger interface with slog implementation and trace correlation"
```

### Task 6: zerolog-реализация + бенчмарки

**Files:**
- Create: `pkg/logger/zerologx.go`
- Test: дополнить `pkg/logger/logger_test.go`, создать `pkg/logger/bench_test.go`

**Interfaces:**
- Consumes: `Logger`, `traceArgs` (Task 5).
- Produces: `func newZeroLogger(cfg *config.Config, out io.Writer) *zeroLogger`.

- [ ] **Step 1: Падающие тесты**

Добавить в `pkg/logger/logger_test.go`:
```go
func TestZeroLoggerWritesJSONInProd(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf)

	log.Info(context.Background(), "hello", "user_id", 42)

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "hello", entry["message"])
	assert.Equal(t, float64(42), entry["user_id"])
}

func TestZeroLoggerLevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf)

	log.Debug(context.Background(), "invisible")

	assert.Empty(t, buf.Bytes())
}

func TestZeroLoggerWith(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf).With("component", "test")

	log.Info(context.Background(), "hello")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "test", entry["component"])
}

func TestZeroLoggerTraceCorrelation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf)

	log.Info(ctxWithSpan(t), "traced")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "01000000000000000000000000000000", entry["trace_id"])
	assert.Equal(t, "0200000000000000", entry["span_id"])
}
```

- [ ] **Step 2: Red**

Run: `go test -count=1 ./pkg/logger/ -v`
Expected: FAIL — `undefined: newZeroLogger`.

- [ ] **Step 3: Зависимость и реализация**

```bash
go get github.com/rs/zerolog@v1.35.1
```

`pkg/logger/zerologx.go`:
```go
package logger

import (
	"context"
	"io"
	"log/slog"

	"clean-arch-template/config"

	"github.com/rs/zerolog"
)

type zeroLogger struct {
	l zerolog.Logger
}

var _ Logger = (*zeroLogger)(nil)

// newZeroLogger — реализация на rs/zerolog с теми же правилами, что slogx:
// prod → JSON в out, иначе — ConsoleWriter; уровень из LOG_LEVEL, DEBUG=true → Debug.
func newZeroLogger(cfg *config.Config, out io.Writer) *zeroLogger {
	if cfg.Environment != "prod" {
		out = zerolog.ConsoleWriter{Out: out}
	}

	l := zerolog.New(out).
		Level(zerologLevel(cfg)).
		With().Timestamp().Logger()

	return &zeroLogger{l: l}
}

func zerologLevel(cfg *config.Config) zerolog.Level {
	if cfg.Debug {
		return zerolog.DebugLevel
	}

	switch {
	case cfg.Level <= slog.LevelDebug:
		return zerolog.DebugLevel
	case cfg.Level <= slog.LevelInfo:
		return zerolog.InfoLevel
	case cfg.Level <= slog.LevelWarn:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

func (z *zeroLogger) Debug(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Debug(), msg, args)
}

func (z *zeroLogger) Info(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Info(), msg, args)
}

func (z *zeroLogger) Warn(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Warn(), msg, args)
}

func (z *zeroLogger) Error(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Error(), msg, args)
}

func (z *zeroLogger) With(args ...any) Logger {
	lctx := z.l.With()
	for k, v := range pairs(args) {
		lctx = lctx.Interface(k, v)
	}

	return &zeroLogger{l: lctx.Logger()}
}

func (z *zeroLogger) log(ctx context.Context, e *zerolog.Event, msg string, args []any) {
	for k, v := range pairs(args) {
		e = e.Interface(k, v)
	}
	for k, v := range pairs(traceArgs(ctx)) {
		e = e.Interface(k, v)
	}
	e.Msg(msg)
}

// pairs итерирует slog-стиль key/value; непарный хвост — под ключом "!BADKEY".
func pairs(args []any) func(yield func(string, any) bool) {
	return func(yield func(string, any) bool) {
		for i := 0; i < len(args); i += 2 {
			if i+1 >= len(args) {
				yield("!BADKEY", args[i])
				return
			}
			key, ok := args[i].(string)
			if !ok {
				key = "!BADKEY"
			}
			if !yield(key, args[i+1]) {
				return
			}
		}
	}
}
```

zerolog по умолчанию пишет message под ключом `message` — совпадает с prod-конвенцией slogx.

- [ ] **Step 4: Green**

Run: `go test -count=1 ./pkg/logger/ -v`
Expected: PASS (9 тестов).

- [ ] **Step 5: Бенчмарки**

`pkg/logger/bench_test.go`:
```go
package logger

import (
	"context"
	"io"
	"testing"

	"clean-arch-template/config"
)

func benchConfig() *config.Config {
	cfg := &config.Config{}
	cfg.App.Environment = "prod"
	return cfg
}

func benchmarkInfo(b *testing.B, log Logger, ctx context.Context) {
	b.Helper()
	b.ReportAllocs()

	for b.Loop() {
		log.Info(ctx, "benchmark message", "user_id", 42, "action", "transfer", "amount", int64(100))
	}
}

func BenchmarkSlogInfo(b *testing.B) {
	benchmarkInfo(b, newSlogLogger(benchConfig(), io.Discard), context.Background())
}

func BenchmarkZerologInfo(b *testing.B) {
	benchmarkInfo(b, newZeroLogger(benchConfig(), io.Discard), context.Background())
}
```

Run: `go test -bench=. -benchmem -run '^$' ./pkg/logger/`
Expected: обе строки с ns/op и allocs/op в выводе (сравнение фиксируется в commit message не требуется).

- [ ] **Step 6: Commit**

```bash
git add pkg/logger/ go.mod go.sum
git commit -m "feat: zerolog implementation with benchmarks"
```

### Task 7: Фабрика, LOG_BACKEND, Nop и fake-логгер

**Files:**
- Modify: `pkg/logger/logger.go` (фабрика + Nop)
- Modify: `config/config.go` (Log.Backend)
- Create: `pkg/logger/loggertest/fake.go`
- Test: дополнить `pkg/logger/logger_test.go`, `config/config_test.go`

**Interfaces:**
- Consumes: `newSlogLogger`, `newZeroLogger` (Tasks 5–6).
- Produces:
  - `func New(cfg *config.Config) (Logger, error)` — фабрика по `cfg.Log.Backend` (`slog` | `zerolog`, ошибка на прочее);
  - `func Nop() Logger` — no-op для дефолтов;
  - `loggertest.Fake` (`Entries []loggertest.Entry{Level, Msg string; Args []any}`, метод `With` возвращает тот же Fake).

- [ ] **Step 1: Падающие тесты**

В `config/config_test.go` (в `TestLoadConfig`):
```go
	assert.Equal(t, "slog", cfg.Log.Backend)
```

В `pkg/logger/logger_test.go`:
```go
func TestNewFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		backend string
		wantErr bool
	}{
		{backend: "slog"},
		{backend: "zerolog"},
		{backend: "syslog", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.backend, func(t *testing.T) {
			t.Parallel()

			cfg := prodConfig()
			cfg.Log.Backend = tc.backend

			log, err := New(cfg)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, log)
		})
	}
}
```

Run: `go test -count=1 ./pkg/logger/ ./config/` — Expected: FAIL (`undefined: New`, `cfg.Log.Backend undefined`).

- [ ] **Step 2: Реализация**

`config/config.go`, struct `Log`:
```go
	Log struct {
		Level   slog.Level `json:"level"   toml:"level"   env:"LOG_LEVEL"`
		Backend string     `json:"backend" toml:"backend" env:"LOG_BACKEND" env-default:"slog"`
	}
```

`pkg/logger/logger.go`, добавить:
```go
const (
	BackendSlog    = "slog"
	BackendZerolog = "zerolog"
)

// New — фабрика логгера по cfg.Log.Backend (env LOG_BACKEND).
func New(cfg *config.Config) (Logger, error) {
	switch cfg.Backend {
	case BackendSlog, "":
		return newSlogLogger(cfg, os.Stdout), nil
	case BackendZerolog:
		return newZeroLogger(cfg, os.Stdout), nil
	default:
		return nil, fmt.Errorf("unknown log backend %q (supported: %s, %s)", cfg.Backend, BackendSlog, BackendZerolog)
	}
}

// Nop — логгер-заглушка для необязательных зависимостей.
func Nop() Logger { return nopLogger{} }

type nopLogger struct{}

func (nopLogger) Debug(context.Context, string, ...any) {}
func (nopLogger) Info(context.Context, string, ...any)  {}
func (nopLogger) Warn(context.Context, string, ...any)  {}
func (nopLogger) Error(context.Context, string, ...any) {}
func (nopLogger) With(...any) Logger                    { return nopLogger{} }
```
(в imports добавить `fmt`, `os`.)

`pkg/logger/loggertest/fake.go`:
```go
// Package loggertest — fake-реализация logger.Logger для юнит-тестов потребителей.
package loggertest

import (
	"context"
	"sync"

	"clean-arch-template/pkg/logger"
)

type Entry struct {
	Level string
	Msg   string
	Args  []any
}

type Fake struct {
	mu      sync.Mutex
	Entries []Entry
}

var _ logger.Logger = (*Fake)(nil)

func (f *Fake) Debug(_ context.Context, msg string, args ...any) { f.record("DEBUG", msg, args) }
func (f *Fake) Info(_ context.Context, msg string, args ...any)  { f.record("INFO", msg, args) }
func (f *Fake) Warn(_ context.Context, msg string, args ...any)  { f.record("WARN", msg, args) }
func (f *Fake) Error(_ context.Context, msg string, args ...any) { f.record("ERROR", msg, args) }
func (f *Fake) With(_ ...any) logger.Logger                      { return f }

func (f *Fake) record(level, msg string, args []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Entries = append(f.Entries, Entry{Level: level, Msg: msg, Args: args})
}
```

- [ ] **Step 3: Green**

Run: `go test -count=1 ./pkg/logger/... ./config/`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add pkg/logger/ config/
git commit -m "feat: logger factory with LOG_BACKEND selection, Nop and test fake"
```

### Task 8: DI — замена глобального slog

**Files:**
- Modify: `cmd/template/main.go`
- Modify: `internal/app/app.go`, `internal/app/migrations.go`, `internal/app/sysinfo.go`
- Modify: `version/version.go`
- Modify: `pkg/tracing/tracing.go`
- Modify: `pkg/database/database.go`, `pkg/database/options.go`, `pkg/database/pool_config.go`
- Modify: `internal/handler/rest/v1/errors.go`, `internal/handler/rest/v1/user.go`
- Modify: `internal/handler/rest/v1/user_test.go`, `internal/handler/rest/v1/converter_test.go` (конструктор)
- Modify: `docker-compose.yml` (проброс LOG_BACKEND)

**Interfaces:**
- Consumes: `logger.New`, `logger.Nop`, `loggertest.Fake` (Task 7).
- Produces (используется финальной верификацией):
  - `app.New(ctx context.Context, cfg *config.Config, log logger.Logger) (*App, error)`
  - `tracing.InitOpenTelemetryGRPC(ctx context.Context, cfg *config.Config, log logger.Logger) (*trace.TracerProvider, error)`
  - `database.WithLogger(l logger.Logger) Option`
  - `v1.NewUserHandler(uc UserUseCase, log logger.Logger) *UserHandler`
  - `version.PrintVersion(cfg *config.Config, log logger.Logger)`, `app.PrintSystemData(log)`, `app.PrintMemoryInfo(log)`

Правило замены вызовов: `slog.X(msg, slog.String(k,v))` → `log.X(ctx, msg, k, v)`; где ctx запроса нет — `context.Background()` (в `main`, стартовых логах) или ctx из сигнатуры (в `Run`, migrations).

- [ ] **Step 1: Падающий тест на логирование 500-х**

В `internal/handler/rest/v1/user_test.go`: helper `newTestAPI` расширить возвратом fake:
```go
func newTestAPI(t *testing.T) (humatest.TestAPI, *loggertest.Fake) {
	t.Helper()

	_, api := humatest.New(t, SetupHumaConfig())

	users := make([]entity.User, len(mockUsers))
	copy(users, mockUsers)

	fakeLog := &loggertest.Fake{}
	mockRepo := &mockUserRepository{users: users}
	userUC := usecase.NewUserUseCase(mockRepo)
	SetupRoutes(api, NewUserHandler(userUC, fakeLog))

	return api, fakeLog
}
```
(во всех существующих тестах — `api, _ := newTestAPI(t)`), и новый тест (mock репозиторий: `GetAllUsers` возвращает `errors.New("boom")` при `limit == 13`, добавить в mockUserRepository):
```go
func TestInternalErrorIsLoggedAndMasked(t *testing.T) {
	api, fakeLog := newTestAPI(t)

	resp := api.Get("/users/1/13") // sentinel-лимит: mock отдаёт internal error
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), "internal server error") || strings.Contains(resp.Body.String(), "boom") {
		t.Fatalf("Internal error text must be masked, got: %s", resp.Body.String())
	}
	if len(fakeLog.Entries) != 1 || fakeLog.Entries[0].Level != "ERROR" {
		t.Fatalf("Expected exactly one ERROR log entry, got %+v", fakeLog.Entries)
	}
}
```
и в `mockUserRepository.GetAllUsers`:
```go
func (m *mockUserRepository) GetAllUsers(_ context.Context, _, limit int) ([]entity.User, error) {
	if limit == 13 {
		return nil, errors.New("boom")
	}
	return m.users, nil
}
```

Run: `go test -count=1 ./internal/handler/rest/v1/` — Expected: FAIL (компиляция: NewUserHandler не принимает логгер).

- [ ] **Step 2: Handler**

`internal/handler/rest/v1/user.go`:
```go
type UserHandler struct {
	userUC UserUseCase
	log    logger.Logger
}

func NewUserHandler(uc UserUseCase, log logger.Logger) *UserHandler {
	return &UserHandler{userUC: uc, log: log}
}
```
`internal/handler/rest/v1/errors.go` — `MapError` становится методом:
```go
// mapError маппит доменные ошибки в HTTP-ошибки. Неизвестные ошибки логируются
// (с trace_id из ctx) и уходят клиенту как generic 500.
func (uh *UserHandler) mapError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUserNotFound),
		errors.Is(err, entity.ErrSourceAccountNotFound),
		errors.Is(err, entity.ErrDestAccountNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, entity.ErrInvalidUserName),
		errors.Is(err, entity.ErrInvalidPagination),
		errors.Is(err, entity.ErrNegativeAmount),
		errors.Is(err, entity.ErrSameAccount):
		return huma.Error400BadRequest(err.Error())
	case errors.Is(err, entity.ErrInsufficientFunds):
		return huma.Error409Conflict(err.Error())
	default:
		uh.log.Error(ctx, "request failed", "error", err.Error())
		return huma.Error500InternalServerError("internal server error")
	}
}
```
Во всех хендлерах `MapError(err)` → `uh.mapError(ctx, err)` (ctx — уже обогащённый span'ом). В `converter_test.go`/`user_test.go` — конструктор с `&loggertest.Fake{}`.

Run: `go test -count=1 ./internal/handler/rest/v1/` — Expected: PASS, включая новый тест.

- [ ] **Step 3: pkg/database опция логгера**

`pkg/database/database.go`: поле `logger logger.Logger` в struct `Postgres`, дефолт `logger.Nop()` в `New`; `slog.Debug(...)` в цикле ретраев → `pg.logger.Debug(context.Background(), ...)`.
`pkg/database/options.go`:
```go
// WithLogger -.
func WithLogger(l logger.Logger) Option {
	return func(c *Postgres) {
		c.logger = l
	}
}
```
`pkg/database/pool_config.go`: `setupPoolConfig(cfg, pg, poolConfig)` использует `pg.logger` вместо глобального slog; структуры `SQLQueryTracer`/`ConnectTracer` получают поле `log logger.Logger` (инициализируются `pg.logger`), вызовы `slog.Debug/Error` → `t.log.Debug(ctx, ...)`/`t.log.Error(ctx, ...)` (ctx из сигнатур Trace-методов; в `TraceQueryEnd/TraceConnectEnd`, где ctx-параметр игнорируется, — `context.Background()`).

- [ ] **Step 4: tracing, version, sysinfo**

`pkg/tracing/tracing.go`: параметр `logger *slog.Logger` → `log logger.Logger`; `logger.With("error", ...).Error(...)` → `log.Error(ctx, "tracing exporter could not be created", "error", err.Error())`.
`version/version.go`: `func PrintVersion(cfg *config.Config, log logger.Logger)`; `slog.Info(fmt.Sprintf(...))` → `log.Info(context.Background(), fmt.Sprintf("Application %s version %s", cfg.Name, Version))`; blank-import `_ "embed"` сохранить.
`internal/app/sysinfo.go`: `func PrintSystemData(log logger.Logger)`, `func PrintMemoryInfo(log logger.Logger)`; все `slog.Info(...)` → `log.Info(context.Background(), ...)`.

- [ ] **Step 5: app и main**

`internal/app/app.go`: `func New(ctx context.Context, cfg *config.Config, log logger.Logger) (*App, error)`; поле `log logger.Logger` в `App`; передать `database.WithLogger(log)` в `database.New`; `applyMigrations(ctx, cfg.DB, log)`; `version.PrintVersion(cfg, log)`, `PrintSystemData(log)`, `PrintMemoryInfo(log)`; в `Run`: `slog.Info` → `a.log.Info(ctx, ...)`.
`internal/app/migrations.go`: `func applyMigrations(ctx context.Context, cfg config.DB, log logger.Logger) error`; все `slog.*` → `log.*(ctx, ...)`.
`cmd/template/main.go`:
```go
	log, err := logger.New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}
```
далее все `slog.Error/Info` → `log.Error(context.Background(), ...)` / `log.Info(context.Background(), ...)`; `tracing.InitOpenTelemetryGRPC(ctx, cfg, log)`; `app.New(ctx, cfg, log)`. Импорт `log/slog` из main уходит.
`docker-compose.yml`, сервис `app`, environment: добавить `LOG_BACKEND: ${LOG_BACKEND}`.

- [ ] **Step 6: Полная сборка и линт**

```bash
go build ./... && go vet ./... && golangci-lint run ./... && grep -rn "log/slog" --include="*.go" cmd/ internal/ pkg/tracing/ pkg/database/ version/
```
Expected: сборка/линт зелёные; grep находит `log/slog` только там, где он легитимен (`pkg/logger/*`, `config/config.go` — тип `slog.Level`). Любое другое вхождение — недоделанная замена.

- [ ] **Step 7: Все юнит-тесты**

Run: `go test -count=1 $(go list ./... | grep -v integration-test)`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add cmd/ internal/ pkg/ version/ docker-compose.yml
git commit -m "feat: inject logger interface everywhere, drop global slog"
```

### Task 9: Smoke оба бэкенда + docs

**Files:**
- Modify: `README.md`, `CLAUDE.md`

**Interfaces:**
- Consumes: всё из Tasks 5–8.

- [ ] **Step 1: Полный стек на slog (дефолт)**

Run: `DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make cleandc`
Expected: интеграционные тесты PASS, `tests exited with code 0`; в логах app видны записи с временем/уровнем.

- [ ] **Step 2: Smoke на zerolog**

Run: `LOG_BACKEND=zerolog DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make dc`
Expected: приложение стартует, `/readyz` → 200 (`curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:9000/readyz`), логи app в zerolog-формате (ConsoleWriter, т.к. ENV_NAME=dev). Остановить стек.

- [ ] **Step 3: Docs**

README Technologies: строку Observability дополнить `общий интерфейс logger.Logger, бэкенды slog | zerolog (LOG_BACKEND), trace_id/span_id в каждой записи с активным спаном`; TODO: отметить `- [x] Общий интерфейс логгера`.
CLAUDE.md, раздел Key conventions, добавить пункт: логирование только через `pkg/logger.Logger` (ctx-first, DI из main; глобальный slog не используется); фабрика `logger.New` по `LOG_BACKEND`; для тестов — `loggertest.Fake`; usecase/repository не логируют.

- [ ] **Step 4: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: logger interface conventions and LOG_BACKEND"
```

---

## Верификация плана против спеки (self-review выполнен)

- Все решения спеки покрыты задачами; отклонение от спеки одно и намеренное: реализации логгера — файлы `pkg/logger/slogx.go`/`zerologx.go` вместо подпакетов (цикл импортов через возвращаемый `With(...) Logger`); fake вынесен в подпакет `loggertest` (цикла нет — импорт односторонний).
- Типы/сигнатуры сквозные: `app.New(ctx, cfg, log)` (Task 8) расширяет `app.New(ctx, cfg)` (Task 3) — при выполнении части 2 после части 1 конфликтов нет, задачи менять местами нельзя.
- `make cleandc`/`make dc` — терминальная проверка каждой части.
