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

	if cfg.Debug == true || cfg.Environment != "prod" {
		opts = &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		}
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	} else if cfg.Environment == "prod" {
		key := func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
				return a
			}
			return a
		}
		opts = &slog.HandlerOptions{ReplaceAttr: key}
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	slog.SetDefault(logger)
}
