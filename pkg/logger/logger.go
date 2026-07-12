package logger

import (
	"clean-arch-template/config"
	"log/slog"
	"os"
)

// SetupLogger настраивает глобальный slog: уровень берётся из конфига
// (LOG_LEVEL), DEBUG=true принудительно опускает его до Debug.
// prod — JSON, иначе — текст с source-позициями.
func SetupLogger(cfg *config.Config) {
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
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: renameMsgKey,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	}

	slog.SetDefault(slog.New(handler))
}
