// Package loggertest provides a fake logger for unit tests (fake logger.Logger for consumer unit tests).
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

type Fake struct {
	mu      sync.Mutex
	Entries []Entry
}

var _ logger.Logger = (*Fake)(nil)

func (f *Fake) Debug(_ context.Context, msg string, args ...any) { f.record("DEBUG", msg, args) }
func (f *Fake) Info(_ context.Context, msg string, args ...any)  { f.record("INFO", msg, args) }
func (f *Fake) Warn(_ context.Context, msg string, args ...any)  { f.record("WARN", msg, args) }
func (f *Fake) Error(_ context.Context, msg string, args ...any) { f.record("ERROR", msg, args) }
func (f *Fake) With(_ ...any) logger.Logger                      { return f }

func (f *Fake) record(level, msg string, args []any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Entries = append(f.Entries, Entry{Level: level, Msg: msg, Args: args})
}
