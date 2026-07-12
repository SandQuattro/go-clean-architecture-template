# Основные принципы “Чистой архитектуры”:

- Независимость от фреймворков: система не должна зависеть от библиотек, что позволяет уменьшить риск ограничений и зависимостей.
- Тестируемость: бизнес-правила могут быть протестированы без UI, базы данных, серверов или любых других внешних элементов.
- Уровни абстракции: разделение кода на слои с четко определенными правилами для перехода от одного к другому.
- Независимость от UI: пользовательский интерфейс можно легко изменить без изменения остальной системы.
- Независимость от баз данных: бизнес-правила не связаны с типом хранения данных.

## Цели и преимущества использования “Чистой архитектуры”
Главная цель “Чистой архитектуры” — создание программного обеспечения, которое легко поддаётся изменениям, расширению функциональности и поддержке.

## Основные компоненты
- Сущности (Entities): объекты предметной области, высокоуровневые бизнес-правила и доменные сентинел-ошибки (`internal/entity`).
- Use Cases: специфичная для приложения бизнес-логика и валидация (`internal/usecase`).
- Контроллеры и презентеры: Huma-хендлеры с транспортными DTO — entity не протекает в публичный контракт (`internal/handler/rest/v1`).
- Внешние интерфейсы (Interface Adapters): репозитории на pgx, преобразование данных для use case (`internal/usecase/repository`).

## Template Technologies
- huma service openapi 3.1 runtime operations generator (закреплён на v2.37.0 — последняя версия с адаптером под fiber v2; v2.38+ требует fiber v3, экосистема которого ещё не готова)
- high performance fiber http server / router / middlewares
- Golang: 1.26+
- fiber middlewares: structured http access logger, panic recovery, resource monitor, pprof profiler, health check (readiness пингует пул БД), request timeout
- Database: Postgres, clean SQL (PGX v5), транзакции через Thiht/transactor, деньги в int64 (минимальные единицы валюты)
- Migrations: goose (pressly/goose v3), применяются автоматически при старте под pg advisory lock; ошибка миграции валит старт
- Config: cleanenv (файл + env поверх, `CONFIG_PATH` для явного пути; таймауты — только env)
- Observability: общий интерфейс `logger.Logger`, бэкенды slog | zerolog (`LOG_BACKEND`, JSON в prod, уровень из `LOG_LEVEL`), trace_id/span_id в каждой записи с активным спаном, Prometheus + Grafana (конфиги в репозитории), OpenTelemetry tracing → Jaeger
- Lint: golangci-lint v2 (`make lint`), gofumpt как форматтер

## Запуск приложения локально
`DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make cleandc`

Поднимает весь стек: app + Postgres (хост-порт **5434**) + Jaeger + Prometheus + Grafana + интеграционные тесты.

## Документация API
[OpenAPI3.1](http://127.0.0.1:9000/docs)

## Наблюдаемость
- [МОНИТОР](http://127.0.0.1:9000/monitor) — cpu, память, время ответа, соединения (dev-only)
- [PROFILER](http://127.0.0.1:9000/debug/pprof/) — CPU, память, горутины, блокировки (dev-only)
- [Jaeger UI](http://localhost:16686) — трейсы
- [Prometheus](http://localhost:9090), [Grafana](http://localhost:3000) (admin/admin)

## Нагрузочное тестирование
`k6 run load-test/load_test.js`

## TODO
- [x] Добавить генерацию моков для интерфейсов
- [x] Добавить tracing на обработчики (span-контекст пробрасывается вниз)
- [x] Добавить unit тесты
- [x] Добавить fuzzing тесты
- [x] Добавить integration тесты (CRUD + transfer)
- [x] Добавить prometheus и grafana в docker-compose
- [x] Добавить unit тесты на слой handler
- [x] Добавить transactor
- [x] Добавить еще интеграционных тестов
- [ ] Привести все тесты в порядок, добавить покрытие > 80%
- [x] Перевести миграции на goose
- [x] Общий интерфейс логгера (slog / zerolog за одной абстракцией)

## Baseline существующей БД (переход с golang-migrate)
Шаблон предполагает свежую БД. Если схема уже создана golang-migrate:
1. Убедитесь, что схема соответствует последней миграции.
2. Создайте таблицу версий goose, выполнив любую goose-команду против этой БД, например `make migrations-status` (упадёт со списком pending — это нормально, таблица `goose_db_version` при этом будет создана).
3. Пометьте каждую версию применённой: `INSERT INTO goose_db_version (version_id, is_applied) VALUES (<version>, true);` — для всех версий из `migrations/` (например, 20230101000000). Пока это не сделано, НЕ запускайте приложение: повторное применение миграции `20260712000001` умножит балансы на 100.
4. Таблица `schema_migrations` от golang-migrate больше не используется и может быть удалена.
