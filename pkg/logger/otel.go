package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// traceArgs достаёт из ctx активный спан; при валидном спане возвращает
// пары атрибутов для корреляции логов с трейсами.
func traceArgs(ctx context.Context) []any {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	return []any{
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	}
}
