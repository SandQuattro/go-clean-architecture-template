// Package tracing otlp
package tracing

import (
	"clean-arch-template/config"
	"clean-arch-template/pkg/logger"
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

const fraction = 0.6

// InitOpenTelemetryGRPC oltp init.
func InitOpenTelemetryGRPC(ctx context.Context, cfg *config.Config, log logger.Logger) (*trace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(cfg.URL),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Error(ctx, "tracing exporter could not be created", "error", err.Error())
		return nil, err
	}

	// labels/tags/resources that are common to all traces.
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.Name),
		// attribute.String("some-attribute", "some-value"), ...
	)

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		// set the sampling rate based on the parent span to 60%
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(fraction))),
	)

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, // W3C Trace Context format; https://www.w3.org/TR/trace-context/
		),
	)

	return provider, nil
}
