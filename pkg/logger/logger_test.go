package logger

import (
	"bytes"
	"clean-arch-template/config"
	"context"
	"encoding/json"
	"testing"

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

func TestZeroLoggerWritesJSONInProd(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf)

	log.Info(context.Background(), "hello", "user_id", 42)

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "hello", entry["message"])
	assert.Equal(t, float64(42), entry["user_id"])
}

func TestZeroLoggerLevelFiltering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf)

	log.Debug(context.Background(), "invisible")

	assert.Empty(t, buf.Bytes())
}

func TestZeroLoggerWith(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf).With("component", "test")

	log.Info(context.Background(), "hello")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "test", entry["component"])
}

func TestZeroLoggerTraceCorrelation(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := newZeroLogger(prodConfig(), &buf)

	log.Info(ctxWithSpan(t), "traced")

	entry := lastJSONLine(t, &buf)
	assert.Equal(t, "01000000000000000000000000000000", entry["trace_id"])
	assert.Equal(t, "0200000000000000", entry["span_id"])
}

func TestNewFactory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		backend string
		wantErr bool
	}{
		{backend: "slog"},
		{backend: "zerolog"},
		{backend: "syslog", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.backend, func(t *testing.T) {
			t.Parallel()

			cfg := prodConfig()
			cfg.Log.Backend = tc.backend

			log, err := New(cfg)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, log)
		})
	}
}

func TestPairs(t *testing.T) {
	t.Parallel()

	collect := func(args []any) [][2]any {
		var got [][2]any
		for k, v := range pairs(args) {
			got = append(got, [2]any{k, v})
		}
		return got
	}

	tests := []struct {
		name string
		args []any
		want [][2]any
	}{
		{name: "empty args yield nothing", args: nil, want: nil},
		{name: "even pairs", args: []any{"a", 1, "b", 2}, want: [][2]any{{"a", 1}, {"b", 2}}},
		{name: "unpaired string tail goes under BADKEY", args: []any{"a", 1, "tail"}, want: [][2]any{{"a", 1}, {"!BADKEY", "tail"}}},
		{name: "non-string key consumes one element like slog", args: []any{42, "a", 1}, want: [][2]any{{"!BADKEY", 42}, {"a", 1}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, collect(tc.args))
		})
	}
}

func TestPairsEarlyExit(t *testing.T) {
	t.Parallel()

	var got [][2]any
	for k, v := range pairs([]any{"a", 1, "b", 2}) {
		got = append(got, [2]any{k, v})
		break
	}

	assert.Equal(t, [][2]any{{"a", 1}}, got)
}

func TestZeroLoggerWithDoesNotMutateOriginal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	base := newZeroLogger(prodConfig(), &buf)
	_ = base.With("component", "child")

	base.Info(context.Background(), "hello")

	entry := lastJSONLine(t, &buf)
	assert.NotContains(t, entry, "component")
}
