package otelx_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/otelx"
)

// spanWith returns a context carrying a span from a provider using sampler.
func spanWith(t *testing.T, sampler sdktrace.Sampler) context.Context {
	t.Helper()

	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sampler))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })

	return ctx
}

func attrMap(attrs []slog.Attr) map[string]string {
	out := make(map[string]string, len(attrs))
	for _, a := range attrs {
		out[a.Key] = a.Value.String()
	}
	return out
}

func TestTraceAttrsWithoutSpan(t *testing.T) {
	t.Parallel()

	assert.Nil(t, otelx.TraceAttrs(context.Background()))
}

func TestTraceAttrsWithSampledSpan(t *testing.T) {
	t.Parallel()

	ctx := spanWith(t, sdktrace.AlwaysSample())

	got := attrMap(otelx.TraceAttrs(ctx))

	sc := trace.SpanContextFromContext(ctx)
	assert.Equal(t, sc.TraceID().String(), got["trace_id"])
	assert.Equal(t, sc.SpanID().String(), got["span_id"])
}

// TestTraceAttrsWithUnsampledSpan pins the reason correlation ids were kept
// alongside trace ids: an unsampled trace reaches no backend, so logging its id
// would offer a link that resolves to nothing. Correlation ids cover that gap.
func TestTraceAttrsWithUnsampledSpan(t *testing.T) {
	t.Parallel()

	ctx := spanWith(t, sdktrace.NeverSample())

	require.True(t, trace.SpanContextFromContext(ctx).IsValid(), "the span context is still valid")
	require.False(t, trace.SpanContextFromContext(ctx).IsSampled())

	assert.Nil(t, otelx.TraceAttrs(ctx), "an unsampled trace must not be logged as a trace id")
}

func TestInitDisabledIsNoop(t *testing.T) {
	t.Parallel()

	shutdown, err := otelx.Init(context.Background(), config.AppConfig{Name: "test"}, config.TracingConfig{Enabled: false})

	require.NoError(t, err)
	require.NotNil(t, shutdown, "a no-op shutdown must still be returned, so callers need no nil check")
	assert.NoError(t, shutdown(context.Background()))
}

// TestInitEnabledDoesNotRequireACollector matters for local runs and CI: the
// OTLP HTTP exporter connects lazily, so start-up must not depend on Tempo
// being reachable.
func TestInitEnabledDoesNotRequireACollector(t *testing.T) {
	t.Parallel()

	cfg := config.TracingConfig{
		Enabled:     true,
		Endpoint:    "127.0.0.1:4318",
		Insecure:    true,
		SampleRatio: 1,
	}

	shutdown, err := otelx.Init(context.Background(), config.AppConfig{Name: "test", Version: "1.0.0"}, cfg)

	require.NoError(t, err)
	require.NotNil(t, shutdown)

	ctx, cancel := context.WithTimeout(context.Background(), 2e9)
	defer cancel()
	assert.NoError(t, shutdown(ctx))
}
