// Package logger — общий интерфейс логгера; реализации (slog, zerolog)
// живут в этом же пакете: вынос в подпакеты создал бы цикл импортов
// интерфейс ↔ фабрика ↔ реализация (метод With возвращает Logger).
package logger

import (
	"clean-arch-template/config"
	"context"
	"log/slog"
	"os"
)

type Logger interface {
	Debug(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
	Warn(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
	// With возвращает логгер с добавленными атрибутами (args — пары key/value).
	With(args ...any) Logger
}

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
