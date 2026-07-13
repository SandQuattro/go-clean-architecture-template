package logger

import (
	"clean-arch-template/config"
	"context"
	"io"
	"log/slog"

	"github.com/rs/zerolog"
)

type zeroLogger struct {
	l zerolog.Logger
}

var _ Logger = (*zeroLogger)(nil)

// newZeroLogger — реализация на rs/zerolog с теми же правилами, что slogx:
// prod → JSON в out, иначе — ConsoleWriter; уровень из LOG_LEVEL, DEBUG=true → Debug.
func newZeroLogger(cfg *config.Config, out io.Writer) *zeroLogger {
	if cfg.Environment != "prod" {
		out = zerolog.ConsoleWriter{Out: out}
	}

	l := zerolog.New(out).
		Level(zerologLevel(cfg)).
		With().Timestamp().Logger()

	return &zeroLogger{l: l}
}

func zerologLevel(cfg *config.Config) zerolog.Level {
	if cfg.Debug {
		return zerolog.DebugLevel
	}

	switch {
	case cfg.Level <= slog.LevelDebug:
		return zerolog.DebugLevel
	case cfg.Level <= slog.LevelInfo:
		return zerolog.InfoLevel
	case cfg.Level <= slog.LevelWarn:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}

func (z *zeroLogger) Debug(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Debug(), msg, args)
}

func (z *zeroLogger) Info(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Info(), msg, args)
}

func (z *zeroLogger) Warn(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Warn(), msg, args)
}

func (z *zeroLogger) Error(ctx context.Context, msg string, args ...any) {
	z.log(ctx, z.l.Error(), msg, args)
}

func (z *zeroLogger) With(args ...any) Logger {
	lctx := z.l.With()
	for k, v := range pairs(args) {
		lctx = lctx.Interface(k, v)
	}

	return &zeroLogger{l: lctx.Logger()}
}

func (z *zeroLogger) log(ctx context.Context, e *zerolog.Event, msg string, args []any) {
	// zerolog возвращает nil-событие для выключенного уровня: выходим сразу,
	// не тратя горячий путь на разбор пар и извлечение спана из ctx.
	if e == nil {
		return
	}

	for k, v := range pairs(args) {
		e = e.Interface(k, v)
	}
	for k, v := range pairs(traceArgs(ctx)) {
		e = e.Interface(k, v)
	}
	e.Msg(msg)
}

// pairs итерирует args в slog-семантике key/value: нестроковый элемент на
// месте ключа уходит значением под "!BADKEY" со сдвигом на один (как в
// log/slog), непарная строка в хвосте — тоже под "!BADKEY". slog.Attr не
// поддерживается: контракт Logger — плоские пары key/value.
func pairs(args []any) func(yield func(string, any) bool) {
	return func(yield func(string, any) bool) {
		for i := 0; i < len(args); {
			key, ok := args[i].(string)
			if !ok {
				if !yield("!BADKEY", args[i]) {
					return
				}
				i++
				continue
			}
			if i+1 >= len(args) {
				yield("!BADKEY", key)
				return
			}
			if !yield(key, args[i+1]) {
				return
			}
			i += 2
		}
	}
}
