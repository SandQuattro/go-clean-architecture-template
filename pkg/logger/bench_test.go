package logger

import (
	"clean-arch-template/config"
	"context"
	"io"
	"testing"

	"go.opentelemetry.io/otel/trace"
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

func benchCtxWithSpan() context.Context {
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x02},
		TraceFlags: trace.FlagsSampled,
	})

	return trace.ContextWithSpanContext(context.Background(), spanCtx)
}

func BenchmarkSlogInfoWithSpan(b *testing.B) {
	benchmarkInfo(b, newSlogLogger(benchConfig(), io.Discard), benchCtxWithSpan())
}

func BenchmarkZerologInfoWithSpan(b *testing.B) {
	benchmarkInfo(b, newZeroLogger(benchConfig(), io.Discard), benchCtxWithSpan())
}
