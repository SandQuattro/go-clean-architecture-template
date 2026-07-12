package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"clean-arch-template/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func prodConfig() *config.Config {
	cfg := &config.Config{}
	cfg.App.Environment = "prod"
	return cfg
}

func ctxWithSpan(t *testing.T) context.Context {
	t.Helper()

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x02},
		TraceFlags: trace.FlagsSampled,
	})
	require.True(t, spanCtx.IsValid())

	return trace.ContextWithSpanContext(context.Background(), spanCtx)
}

func lastJSONLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))

	return entry
}

func TestSlogLoggerWritesJSONInProd(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf)

	log.Info(context.Background(), "hello", "user_id", 42)

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "hello", entry["message"])
	assert.Equal(t, float64(42), entry["user_id"])
}

func TestSlogLoggerLevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf) // prod default: Info

	log.Debug(context.Background(), "invisible")

	assert.Empty(t, buf.Bytes())
}

func TestSlogLoggerWith(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf).With("component", "test")

	log.Info(context.Background(), "hello")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "test", entry["component"])
}

func TestSlogLoggerTraceCorrelation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newSlogLogger(prodConfig(), &buf)

	log.Info(ctxWithSpan(t), "traced")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "01000000000000000000000000000000", entry["trace_id"])
	assert.Equal(t, "0200000000000000", entry["span_id"])
}

func TestTraceArgsWithoutSpan(t *testing.T) {
	t.Parallel()

	assert.Nil(t, traceArgs(context.Background()))
}

func TestSlogLoggerSourcePointsAtCaller(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	cfg.App.Environment = "dev"

	var buf bytes.Buffer
	log := newSlogLogger(cfg, &buf)

	log.Info(context.Background(), "hello")

	require.Contains(t, buf.String(), "logger_test.go", "source должен указывать на вызывающий файл, а не на slogx.go")
	require.NotContains(t, buf.String(), "slogx.go")
}
