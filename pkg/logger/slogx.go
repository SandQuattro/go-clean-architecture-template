package logger

import (
	"context"
	"io"
	"log/slog"
	"runtime"
	"time"

	"clean-arch-template/config"
)

type slogLogger struct {
	l *slog.Logger
}

var _ Logger = (*slogLogger)(nil)

// newSlogLogger — реализация на stdlib slog: prod → JSON с ключом message,
// иначе text с source-позициями; уровень из LOG_LEVEL, DEBUG=true → Debug.
func newSlogLogger(cfg *config.Config, out io.Writer) *slogLogger {
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
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: renameMsgKey,
		})
	} else {
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	}

	return &slogLogger{l: slog.New(handler)}
}

func (s *slogLogger) Debug(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelDebug, msg, args)
}

func (s *slogLogger) Info(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelInfo, msg, args)
}

func (s *slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelWarn, msg, args)
}

func (s *slogLogger) Error(ctx context.Context, msg string, args ...any) {
	s.log(ctx, slog.LevelError, msg, args)
}

func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.l.With(args...)}
}

func (s *slogLogger) log(ctx context.Context, level slog.Level, msg string, args []any) {
	if !s.l.Enabled(ctx, level) {
		return
	}

	// Пишем запись через Handler с PC вызывающего кода: прямой вызов
	// s.l.Log добавил бы фреймы обёртки и сломал AddSource (source
	// указывал бы на этот файл, а не на место вызова).
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip: Callers, log, экспортируемый метод (Debug/Info/...)

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	if tr := traceArgs(ctx); tr != nil {
		args = append(append(make([]any, 0, len(args)+len(tr)), args...), tr...)
	}
	r.Add(args...)

	_ = s.l.Handler().Handle(ctx, r)
}
