package logger

import (
	"context"
	"io"
	"testing"

	"clean-arch-template/config"
)

func benchConfig() *config.Config {
	cfg := &config.Config{}
	cfg.App.Environment = "prod"
	return cfg
}

func benchmarkInfo(b *testing.B, log Logger, ctx context.Context) {
	b.Helper()
	b.ReportAllocs()

	for b.Loop() {
		log.Info(ctx, "benchmark message", "user_id", 42, "action", "transfer", "amount", int64(100))
	}
}

func BenchmarkSlogInfo(b *testing.B) {
	benchmarkInfo(b, newSlogLogger(benchConfig(), io.Discard), context.Background())
}

func BenchmarkZerologInfo(b *testing.B) {
	benchmarkInfo(b, newZeroLogger(benchConfig(), io.Discard), context.Background())
}
