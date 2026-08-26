// Package otelx wires OpenTelemetry tracing and exposes the bridge between a
// span and a log record.
//
// It is the only place in the module that imports the OpenTelemetry SDK, so
// tracing stays removable and slogging stays dependency free.
package otelx

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/softika/gopherizer/config"
)

// Shutdown flushes pending spans and releases the exporter.
type Shutdown func(context.Context) error

// noopShutdown is returned when tracing is disabled, so callers never have to
// nil-check before deferring it.
func noopShutdown(context.Context) error { return nil }

// Init installs the global tracer provider and propagator.
//
// When tracing is disabled it installs nothing: OpenTelemetry's default global
// provider is already a no-op, so instrumented code keeps compiling and running
// at effectively no cost.
func Init(ctx context.Context, app config.AppConfig, cfg config.TracingConfig) (Shutdown, error) {
	if !cfg.Enabled {
		return noopShutdown, nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	// The exporter connects lazily, so an unreachable collector does not stop
	// the process from starting.
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp trace exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(app.Name),
		semconv.ServiceVersion(app.Version),
		semconv.DeploymentEnvironmentNameKey.String(app.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to build trace resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// ParentBased keeps a trace whole: once an upstream service decides to
		// sample, every downstream hop follows that decision instead of
		// re-rolling and leaving holes in the trace.
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)

	otel.SetTracerProvider(provider)

	// Route the SDK's internal errors to slog at Error level.
	//
	// Without this they go through the standard library logger, which
	// slog.SetDefault funnels into the handler at Info -- so a collector that
	// is refusing every export reads as routine information. Telemetry that
	// fails quietly is worse than no telemetry, because it is still trusted.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		slog.Error("opentelemetry error", "error", err)
	}))

	// W3C trace context is what carries a trace across services. Baggage is
	// included so callers can propagate their own key/values.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}

// TraceAttrs is a slogging.Extractor that puts the active trace onto a record.
//
// Only sampled traces are reported. An unsampled trace never reaches the
// backend, so logging its id would offer a link that resolves to nothing --
// which is why correlation ids are kept alongside these rather than replaced by
// them: correlation ids are present on every request, sampled or not.
func TraceAttrs(ctx context.Context) []slog.Attr {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() || !sc.IsSampled() {
		return nil
	}

	return []slog.Attr{
		slog.String("trace_id", sc.TraceID().String()),
		slog.String("span_id", sc.SpanID().String()),
	}
}
