// Package logger — общий интерфейс логгера; реализации (slog, zerolog)
// живут в этом же пакете: вынос в подпакеты создал бы цикл импортов
// интерфейс ↔ фабрика ↔ реализация (метод With возвращает Logger).
package logger

import (
	"clean-arch-template/config"
	"context"
	"fmt"
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
