# Дизайн: миграции на goose + общий интерфейс логгера (slog / zerolog)

Дата: 2026-07-12
Статус: одобрен пользователем

## Контекст

Сейчас миграции выполняет golang-migrate (парные `.up.sql`/`.down.sql`, применяются на старте из `internal/app/migrations.go`, таблица `schema_migrations`). Логирование — глобальный `slog` (`pkg/logger.SetupLogger`); после архитектурного ревью логируют только `main`, `internal/app`, `pkg/*` и `handler.MapError`; бизнес-слои (usecase, repository) не логируют.

Две независимые задачи:
1. Перевести миграции на goose (github.com/pressly/goose/v3 **v3.27.2**).
2. Ввести собственный интерфейс логгера с подставляемыми реализациями slog и zerolog (**v1.35.1**).

## Решения, принятые с пользователем

- Логгер: **ctx-first методы** — каждая запись обогащается `trace_id`/`span_id` из OTel-спана.
- Выбор бэкенда логгера: **env-конфиг** `LOG_BACKEND=slog|zerolog` (default `slog`).
- Миграции: **файлы с диска** (не embed.FS) — можно менять без пересборки; Dockerfile продолжает копировать `migrations/`.

## Часть 1: goose

### Зависимости
- `+ github.com/pressly/goose/v3 v3.27.2`
- `− github.com/golang-migrate/migrate/v4` (из go.sum уходит и lib/pq)
- Для подключения мигратора используется `database/sql` поверх pgx: `github.com/jackc/pgx/v5/stdlib` (уже в дереве зависимостей).

### internal/app/migrations.go
```go
db, err := sql.Open("pgx", cfg.DSN())          // короткоживущее соединение только для миграций
provider, err := goose.NewProvider(
    goose.DialectPostgres,
    db,
    os.DirFS(cfg.MigrationsDir),
    goose.WithSessionLocker(sessionLocker), // lock.NewPostgresSessionLocker(...) (SessionLocker, error)
)
results, err := provider.Up(ctx)
```
- Ретраи подключения (3 попытки с паузой) и пропагация ошибок наверх — как сейчас: ошибка миграции валит старт.
- Session-lock: `github.com/pressly/goose/v3/lock`, `NewPostgresSessionLocker(opts ...SessionLockerOption) (SessionLocker, error)` — pg advisory lock защищает от параллельного применения при нескольких репликах (сигнатура сверена с v3.27.2).
- `db.Close()` и `provider.Close()` после применения.

### Конфиг
- `DB.MigrationsDir string` — env `MIGRATIONS_DIR`, env-default `migrations`.

### Конвертация миграций
Пары `.up.sql`/`.down.sql` → одиночные `.sql` с секциями:
```sql
-- +goose Up
<up-SQL>

-- +goose Down
<down-SQL>
```
- Числовые префиксы имён сохраняются (goose принимает числовые версии): `20230101000000_create_users_table.sql` и т.д. (4 миграции).
- Multi-statement SQL не требует `StatementBegin/End` — в файлах нет функций/процедур с `;` внутри.

### Таблица версий и совместимость
- goose ведёт `goose_db_version`; `schema_migrations` от golang-migrate игнорируется.
- Шаблон предполагает свежую БД (`make cleandc`).
- Для существующей БД — baseline-процедура в README: убедиться, что схема соответствует последней миграции, затем `./bin/goose -dir migrations postgres "<DSN>" up-to <version> --no-versioning=false` либо вручную заполнить `goose_db_version` (описать одной командой `goose ... version`/`INSERT`). Точную команду зафиксировать при реализации.

### Makefile
- Установка goose CLI v3.27.2 в `./bin` (аналогично прочим инструментам).
- `make new-migration name=...` → `$(GOOSE) -dir ./migrations create $(name) sql`.
- `make migrations-status` → `$(GOOSE) -dir ./migrations postgres "$(DSN)" status`.

### Прочее
- `cmd/template/main.go`: убрать blank-import `github.com/golang-migrate/migrate/v4/database/postgres`.
- Dockerfile: без изменений (COPY migrations остаётся).

## Часть 2: интерфейс логгера

### Интерфейс (pkg/logger)
```go
type Logger interface {
    Debug(ctx context.Context, msg string, args ...any) // args — slog-стиль key/value
    Info(ctx context.Context, msg string, args ...any)
    Warn(ctx context.Context, msg string, args ...any)
    Error(ctx context.Context, msg string, args ...any)
    With(args ...any) Logger
}
```
- Для записей вне запроса передаётся `context.Background()`.

### Реализации
- `pkg/logger/slogx` — обёртка `*slog.Logger`: JSON в prod (`message` вместо `msg`), text + AddSource в dev; уровень из `LOG_LEVEL`, `DEBUG=true` опускает до Debug (текущее поведение сохраняется).
- `pkg/logger/zerologx` — zerolog v1.35.1 с теми же правилами уровня/формата (prod → JSON stdout; dev → `zerolog.ConsoleWriter`).
- Общий хелпер (например, `pkg/logger/internal/otelattrs` или функция в pkg/logger): из ctx достаёт `trace.SpanContext`; если валиден — добавляет `trace_id`, `span_id` к записи. Используется обеими реализациями.

### Фабрика и конфиг
- `logger.New(cfg *config.Config) Logger` — по `cfg.Log.Backend`.
- `Log.Backend string` — env `LOG_BACKEND`, env-default `slog`; неизвестное значение — ошибка конфигурации при старте.

### DI (глобальный slog уходит из нашего кода)
| Точка | Изменение |
|---|---|
| `cmd/template/main.go` | создаёт Logger фабрикой, логирует фатальные/финальные события |
| `internal/app.New/Run` | принимает Logger, передаёт вниз |
| `internal/app/migrations.go` | логирует через Logger |
| `internal/app/sysinfo.go`, `version.PrintVersion` | принимают Logger параметром |
| `pkg/database` | debug-трейсеры пула (`pool_config.go`) получают Logger через опцию `database.WithLogger(l)` |
| `pkg/tracing` | принимает Logger вместо `*slog.Logger` |
| `internal/handler/rest/v1` | `NewUserHandler(uc, log)`; `MapError` → метод `mapError(ctx, err)` — 500-е логируются с trace_id |
- usecase/repository не логируют — не меняются.
- Fiber access-логгер остаётся отдельным middleware (вне интерфейса).
- `slog.SetDefault` больше не вызывается: сторонние библиотеки, пишущие в глобальный slog, остаются на дефолтном хендлере stdlib.

### Тестирование
- Fake Logger (`pkg/logger/loggertest`, записывает уровень/сообщение/атрибуты) — для юнитов потребителей (например, `mapError` логирует 500-е).
- Table-driven тесты обоих адаптеров: уровни фильтруются, `With` наследуется, `trace_id`/`span_id` появляются при ctx со спаном (recorder-экспортер OTel SDK), формат prod/dev.
- Бенчмарки адаптеров (горячий путь): `Info` с 3 атрибутами, с ctx без спана и со спаном; slogx vs zerologx.

## Error handling
- Миграции: любая ошибка (подключение, парсинг, apply, dirty) — `return err`, приложение не стартует.
- Фабрика логгера: неизвестный `LOG_BACKEND` — ошибка старта.

## Порядок реализации и верификация
1. **Шаг 1 — goose** (изолированный): зависимости, конвертация файлов, migrations.go, конфиг, Makefile; юнит-сборка + полный `make cleandc` (миграции на чистом Postgres + интеграционные тесты). Отдельный коммит.
2. **Шаг 2 — логгер**: интерфейс, две реализации, фабрика, DI по таблице выше, тесты и бенчмарки; `make cleandc` со `LOG_BACKEND=zerolog` для smoke-проверки второго бэкенда. Отдельный коммит (или два: реализация + тесты).
3. Обновить CLAUDE.md и README (Technologies: goose вместо golang-migrate; LOG_BACKEND; отметить TODO-пункты).

## Не в объёме (non-goals)
- Логирование в usecase/repository (по ревью там его быть не должно).
- Embed миграций в бинарь (решение пользователя — файлы).
- Замена fiber access-логгера.
- Down-миграции в рантайме приложения (только через goose CLI).
