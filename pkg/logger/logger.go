package logger

import (
	"log/slog"
	"os"

	"clean-arch-template/config"
)

func SetupLogger(cfg *config.Config) {
	var opts *slog.HandlerOptions
	var logger *slog.Logger

	slog.SetLogLoggerLevel(cfg.Level)

	if os.Getenv("DEBUG") == "true" || os.Getenv("ENV_NAME") != "prod" {
		opts = &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	} else if os.Getenv("ENV_NAME") == "prod" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	slog.SetDefault(logger)
}
