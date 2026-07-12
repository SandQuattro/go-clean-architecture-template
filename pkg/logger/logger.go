// Package logger — общий интерфейс логгера; реализации (slog, zerolog)
// живут в этом же пакете: вынос в подпакеты создал бы цикл импортов
// интерфейс ↔ фабрика ↔ реализация (метод With возвращает Logger).
package logger

import (
	"clean-arch-template/config"
	"context"
	"fmt"
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

const (
	BackendSlog    = "slog"
	BackendZerolog = "zerolog"
)

// New — фабрика логгера по cfg.Log.Backend (env LOG_BACKEND).
func New(cfg *config.Config) (Logger, error) {
	switch cfg.Backend {
	case BackendSlog, "":
		return newSlogLogger(cfg, os.Stdout), nil
	case BackendZerolog:
		return newZeroLogger(cfg, os.Stdout), nil
	default:
		return nil, fmt.Errorf("unknown log backend %q (supported: %s, %s)", cfg.Backend, BackendSlog, BackendZerolog)
	}
}

// Nop — логгер-заглушка для необязательных зависимостей.
func Nop() Logger { return nopLogger{} }

type nopLogger struct{}

func (nopLogger) Debug(context.Context, string, ...any) {}
func (nopLogger) Info(context.Context, string, ...any)  {}
func (nopLogger) Warn(context.Context, string, ...any)  {}
func (nopLogger) Error(context.Context, string, ...any) {}
func (nopLogger) With(...any) Logger                    { return nopLogger{} }

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
