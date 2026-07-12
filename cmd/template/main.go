package main

import (
	"clean-arch-template/config"
	"clean-arch-template/internal/app"
	"clean-arch-template/pkg/logger"
	"clean-arch-template/pkg/tracing"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

func main() {
	// Load environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize the logger
	logger.SetupLogger(cfg)

	if err := run(cfg); err != nil {
		slog.Error("application terminated", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("Server gracefully stopped, bye, bye!")
}

func run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tracerProvider, err := tracing.InitOpenTelemetryGRPC(ctx, cfg, slog.Default())
	if err != nil {
		return err
	}

	// Трейсер гасится после остановки сервера (defer выполняется последним),
	// чтобы не потерять спаны запросов, дренированных при shutdown.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown tracer provider", slog.String("error", err.Error()))
		}
	}()

	application, err := app.New(cfg)
	if err != nil {
		return err
	}

	// Блокируемся до сигнала или ошибки сервера: ошибки старта и работы
	// больше не теряются в горутине.
	return application.Run(ctx)
}
