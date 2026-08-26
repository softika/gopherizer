package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerName identifies this instrumentation in the exported telemetry.
const tracerName = "github.com/softika/gopherizer/database"

// Option adjusts the pool configuration before the pool is opened.
type Option func(*pgxpool.Config)

// WithQueryTracer records a span for every query the pool executes.
//
// Instrumenting at the driver means no repository has to know about tracing:
// every query, including those inside a transaction, is covered by construction
// rather than by remembering to annotate each call site.
func WithQueryTracer(tp trace.TracerProvider) Option {
	return func(c *pgxpool.Config) {
		c.ConnConfig.Tracer = newQueryTracer(tp)
	}
}

// queryTracer implements pgx.QueryTracer.
type queryTracer struct {
	tracer trace.Tracer
}

func newQueryTracer(tp trace.TracerProvider) queryTracer {
	return queryTracer{tracer: tp.Tracer(tracerName)}
}

// TraceQueryStart opens a span for the query.
//
// The statement is recorded; its arguments deliberately are not. Statements are
// parameterised, so the text carries no values, while data.Args holds the real
// ones -- names, emails, tokens. Spans leave this process for a third-party
// backend, and that data must not go with them.
func (t queryTracer) TraceQueryStart(
	ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData,
) context.Context {
	ctx, _ = t.tracer.Start(ctx, "pg.query",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemNamePostgreSQL,
			semconv.DBQueryText(data.SQL),
		),
	)

	return ctx
}

// TraceQueryEnd closes the span opened by TraceQueryStart.
func (t queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := trace.SpanFromContext(ctx)
	defer span.End()

	// ErrNoRows is how a lookup reports "not found". That is an ordinary
	// outcome, and marking it a failure would bury the real ones.
	if data.Err != nil && !errors.Is(data.Err, pgx.ErrNoRows) {
		span.SetStatus(codes.Error, data.Err.Error())
	}
}
