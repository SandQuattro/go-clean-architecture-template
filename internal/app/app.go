package app

import (
	"clean-arch-template/config"
	"clean-arch-template/internal/usecase"
	"clean-arch-template/internal/usecase/repository"
	"clean-arch-template/pkg/database"
	"clean-arch-template/pkg/logger"
	"clean-arch-template/version"
	"context"
	"fmt"

	v1 "clean-arch-template/internal/handler/rest/v1"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humafiber"
	"github.com/gofiber/fiber/v2/middleware/adaptor"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type App struct {
	server *fiber.App
	pg     *database.Postgres
	cfg    *config.Config
	log    logger.Logger
}

// New подключает БД, применяет миграции, собирает middleware и DI.
// Любая ошибка старта возвращается наверх — приложение не должно жить
// с недоступной БД или битой схемой.
func New(ctx context.Context, cfg *config.Config, log logger.Logger) (*App, error) {
	//nolint:contextcheck // стартовый лог до появления запроса: сигнатура фиксирована без ctx
	version.PrintVersion(cfg, log)

	//nolint:contextcheck // database.New не принимает ctx: пул создаётся один раз при старте
	pg, err := database.New(cfg,
		database.MaxPoolSize(cfg.PoolMax),
		database.MinPoolSize(cfg.PoolMin),
		database.ConnTimeout(cfg.ConnectTimeout),
		database.HealthCheckPeriod(cfg.HealthCheckPeriod),
		database.WithLogger(log),
	)
	if err != nil {
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}

	if err := applyMigrations(ctx, cfg.DB, log); err != nil {
		pg.Close()
		return nil, fmt.Errorf("apply migrations failed: %w", err)
	}

	// ReadTimeout обязателен: без него fiber не закрывает keepalive-соединения
	// при Shutdown и drain никогда не завершается.
	server := fiber.New(fiber.Config{
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	})

	//nolint:contextcheck // per-request ctx внутри middleware намеренно независим от ctx старта приложения
	setupMiddlewares(server, cfg, pg)
	setupRoutes(server, pg, log)

	//nolint:contextcheck // стартовые диагностические логи: сигнатура фиксирована без ctx
	PrintSystemData(log)
	//nolint:contextcheck // стартовые диагностические логи: сигнатура фиксирована без ctx
	PrintMemoryInfo(log)

	return &App{server: server, pg: pg, cfg: cfg, log: log}, nil
}

// Run блокируется до отмены контекста (сигнал) или ошибки сервера.
// При отмене выполняет graceful shutdown с таймаутом и закрывает пул БД.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- a.server.Listen(":" + a.cfg.Port)
	}()

	a.log.Info(ctx, "Starting server on port: "+a.cfg.Port)

	select {
	case err := <-errCh:
		a.pg.Close()
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	// Свежий контекст: родительский уже отменён, а drain должен получить
	// полноценный таймаут.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
	defer cancel()

	//nolint:contextcheck // shutdown-контекст сознательно не наследует отменённый родительский
	err := a.server.ShutdownWithContext(shutdownCtx)
	a.pg.Close()
	if err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	a.log.Info(ctx, "Server stopped")

	return nil
}

func setupMiddlewares(server *fiber.App, cfg *config.Config, pg *database.Postgres) {
	if cfg.Environment == "prod" {
		// Структурированный access-лог, чтобы не ломать JSON-пайплайн логов.
		server.Use(fiberlogger.New(fiberlogger.Config{
			Format: `{"time":"${time}","message":"access","method":"${method}","path":"${path}","status":${status},"latency":"${latency}","ip":"${ip}"}` + "\n",
		}))
	} else {
		server.Use(fiberlogger.New())
	}

	// open telemetry
	server.Use(otelfiber.Middleware())

	// readiness отражает реальную готовность: умерла БД — /readyz отдаёт 503.
	server.Use(healthcheck.New(healthcheck.Config{
		ReadinessProbe: func(c *fiber.Ctx) bool {
			return pg.Pool.Ping(c.UserContext()) == nil
		},
	}))

	server.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Таймаут на запрос: зависший запрос к БД не держит соединение пула вечно.
	server.Use(func(c *fiber.Ctx) error {
		reqCtx, cancel := context.WithTimeout(c.UserContext(), cfg.RequestTimeout)
		defer cancel()
		c.SetUserContext(reqCtx)
		return c.Next()
	})

	// go metrics
	server.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	// fiber metrics
	prometheus := fiberprometheus.New("clean-arch-template")
	prometheus.RegisterAt(server, "/fiber")
	prometheus.SetSkipPaths([]string{"/ping"}) // Optional: Remove some paths from metrics
	server.Use(prometheus.Middleware)

	if cfg.Environment == "dev" {
		server.Use(pprof.New())
		server.Get("/monitor", monitor.New())
	}

	server.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.Status(fiber.StatusOK).SendString("OK")
	})
}

func setupRoutes(server *fiber.App, pg *database.Postgres, log logger.Logger) {
	humaConfig := v1.SetupHumaConfig()
	api := humafiber.New(server, humaConfig)

	// Initialize use cases
	userUseCase := usecase.NewUserUseCase(repository.NewUserRepository(pg.DBGetter, pg.Transactor))

	// Initialize handlers
	userHandler := v1.NewUserHandler(userUseCase, log)
	v1.SetupRoutes(api, userHandler)
}
