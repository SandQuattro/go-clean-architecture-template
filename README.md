# Основные принципы “Чистой архитектуры”:

- Независимость от фреймворков: система не должна зависеть от библиотек, что позволяет уменьшить риск ограничений и зависимостей.
- Тестируемость: бизнес-правила могут быть протестированы без UI, базы данных, серверов или любых других внешних элементов.
- Уровни абстракции: разделение кода на слои с четко определенными правилами для перехода от одного к другому.
- Независимость от UI: пользовательский интерфейс можно легко изменить без изменения остальной системы.
- Независимость от баз данных: бизнес-правила не связаны с типом хранения данных.

## Цели и преимущества использования “Чистой архитектуры” в разработке программного обеспечения
Главная цель “Чистой архитектуры” — создание такого программного обеспечения, которое легко поддается изменениям, расширению функциональности и поддержке.

## Основные компоненты “Чистой архитектуры”
- Сущности (Entities): представляют объекты предметной области с высокоуровневыми правилами бизнеса.
- Use Cases: содержат специфичную для приложения бизнес-логику.
- Контроллеры и презентеры: служат связующим звеном между использованием use case и пользовательским интерфейсом или другими методами доставки информации (например, API).
- Внешние интерфейсы (Interface Adapters): преобразуют данные к формату удобному для Use Cases или сущностей.

## Template Technologies
- huma service openapi 3.1 runtime operations generator
- high performance fiber http server / router / middlewares
- Golang: 1.22+
- fiber middlewares: http access logger, panic recovery, resource monitor, pprof profiler, health check
- Database: Postgres, clean SQL(PGX либа), самый быстрый вариант
- Migrations: golang-migrate
- Config: cleanenv
- Observability: Logging slog (stdlib) json(production), Metrics, OpenTelemetry tracing, стандартный observability стек

##  Запуск приложения локально
DEBUG=true DB_PASSWORD=admin ENV_NAME=dev make cleandc

## Документация API
[OpenAPI3.1](http://127.0.0.1:9000/docs)

## Мониторинг потребления ресурсов
- потребление cpu
- текущее потребление памяти процессом / ОС / общее количество памяти OC
- среднее время ответа на запрос
- количество открытых соединений

[МОНИТОР](http://127.0.0.1:9000/monitor)

## Встроенный профайлер, с поддержкой профилирования CPU, памяти, горутин и блокировок
[PROFILER](http://127.0.0.1:9000/debug/pprof/)

## Нагрузочное тестирование
`
Запускаем разгон до N пользователей, держим N пользователей M секунд, плавно опускаем нагрузку до 0
k6 run load-test/load_test.js
`

## TODO
- [x] Добавить генерацию моков для интерфейсов, чтобы затем использовать их в интеграционных тестах
- [x] Добавить tracing на обработчики
- [x] Добавить unit тесты
- [x] Добавить fuzzing тесты
- [x] Добавить integration тесты
- [x] Добавить prometheus и grafana в docker-compose
- [x] Добавить unit тесты на слой handler
- [ ] Добавить transactor
- [ ] Добавить еще интеграционных тестов
- [ ] Привести все тесты в порядок, добавить покрытие > 80%
