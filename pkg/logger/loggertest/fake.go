// Package loggertest provides a fake logger.Logger implementation for consumer unit tests.
package loggertest

import (
	"clean-arch-template/pkg/logger"
	"context"
	"sync"
)

type Entry struct {
	Level string
	Msg   string
	Args  []any
}

// Fake накапливает записи для ассертов. With возвращает дочерний Fake:
// он пишет в Entries корневого, добавляя накопленные With-атрибуты
// в начало Args каждой записи.
type Fake struct {
	mu      sync.Mutex
	Entries []Entry

	root *Fake
	with []any
}

var _ logger.Logger = (*Fake)(nil)

func (f *Fake) Debug(_ context.Context, msg string, args ...any) { f.record("DEBUG", msg, args) }
func (f *Fake) Info(_ context.Context, msg string, args ...any)  { f.record("INFO", msg, args) }
func (f *Fake) Warn(_ context.Context, msg string, args ...any)  { f.record("WARN", msg, args) }
func (f *Fake) Error(_ context.Context, msg string, args ...any) { f.record("ERROR", msg, args) }

func (f *Fake) With(args ...any) logger.Logger {
	combined := make([]any, 0, len(f.with)+len(args))
	combined = append(combined, f.with...)
	combined = append(combined, args...)

	return &Fake{root: f.rootFake(), with: combined}
}

func (f *Fake) record(level, msg string, args []any) {
	all := make([]any, 0, len(f.with)+len(args))
	all = append(all, f.with...)
	all = append(all, args...)

	r := f.rootFake()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Entries = append(r.Entries, Entry{Level: level, Msg: msg, Args: all})
}

func (f *Fake) rootFake() *Fake {
	if f.root != nil {
		return f.root
	}

	return f
}
