package main

import (
	"clean-arch-template/config"
	"clean-arch-template/internal/app"
	"clean-arch-template/pkg/logger"
	"clean-arch-template/pkg/tracing"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Load environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize the logger
	log, err := logger.New(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger: %v", err))
	}

	if err := run(cfg, log); err != nil {
		log.Error(context.Background(), "application terminated", "error", err.Error())
		os.Exit(1)
	}

	log.Info(context.Background(), "Server gracefully stopped, bye, bye!")
}

func run(cfg *config.Config, log logger.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tracerProvider, err := tracing.InitOpenTelemetryGRPC(ctx, cfg, log)
	if err != nil {
		return err
	}

	// Трейсер гасится после остановки сервера (defer выполняется последним),
	// чтобы не потерять спаны запросов, дренированных при shutdown.
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			log.Error(context.Background(), "failed to shutdown tracer provider", "error", err.Error())
		}
	}()

	application, err := app.New(ctx, cfg, log)
	if err != nil {
		return err
	}

	// Блокируемся до сигнала или ошибки сервера: ошибки старта и работы
	// больше не теряются в горутине.
	return application.Run(ctx)
}
