package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func recordingProvider(t *testing.T) (trace.TracerProvider, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	return tp, exporter
}

func TestQueryTracerRecordsSpan(t *testing.T) {
	t.Parallel()

	tp, exporter := recordingProvider(t)
	tracer := newQueryTracer(tp)

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL: "SELECT id FROM profiles WHERE id = $1",
	})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	assert.Equal(t, trace.SpanKindClient, spans[0].SpanKind)
	assert.Equal(t, codes.Unset, spans[0].Status.Code)
}

// TestQueryTracerNeverRecordsArguments is a privacy guard, not a nicety.
//
// Spans leave the process for a third-party backend. The statement is
// parameterised so it carries no values, but Args holds the real ones -- names,
// emails, tokens -- and must never be attached.
func TestQueryTracerNeverRecordsArguments(t *testing.T) {
	t.Parallel()

	tp, exporter := recordingProvider(t)
	tracer := newQueryTracer(tp)

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL:  "SELECT id FROM profiles WHERE email = $1",
		Args: []any{"someone@example.com", "hunter2"},
	})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	for _, attr := range spans[0].Attributes {
		value := attr.Value.String()
		assert.NotContains(t, value, "someone@example.com", "query argument leaked into attribute %q", attr.Key)
		assert.NotContains(t, value, "hunter2", "query argument leaked into attribute %q", attr.Key)
	}
}

func TestQueryTracerRecordsStatementText(t *testing.T) {
	t.Parallel()

	tp, exporter := recordingProvider(t)
	tracer := newQueryTracer(tp)

	const sql = "SELECT id FROM profiles WHERE id = $1"

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: sql})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	var found bool
	for _, attr := range spans[0].Attributes {
		if attr.Value.String() == sql {
			found = true
		}
	}
	assert.True(t, found, "the parameterised statement should be recorded")
}

func TestQueryTracerMarksFailures(t *testing.T) {
	t.Parallel()

	tp, exporter := recordingProvider(t)
	tracer := newQueryTracer(tp)

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errors.New("boom")})

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	assert.Equal(t, codes.Error, spans[0].Status.Code)
}

// TestQueryTracerIgnoresNoRows keeps traces honest: pgx.ErrNoRows is how a
// lookup reports "not found", which is an ordinary outcome and not a fault.
func TestQueryTracerIgnoresNoRows(t *testing.T) {
	t.Parallel()

	tp, exporter := recordingProvider(t)
	tracer := newQueryTracer(tp)

	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: pgx.ErrNoRows})

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	assert.Equal(t, codes.Unset, spans[0].Status.Code, "a missing row is not a failed query")
}
