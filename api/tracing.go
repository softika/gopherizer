package api

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// tracerName identifies this instrumentation in the exported telemetry.
const tracerName = "github.com/softika/gopherizer/api"

// tracing starts a server span for every request and joins it to an upstream
// trace when the caller sent one.
//
// The provider and propagator are injected rather than read from OpenTelemetry's
// globals, so a test can assert on real spans without mutating process-wide
// state that every other test shares.
//
// This is deliberately hand-written rather than otelhttp: otelhttp names the
// span before chi has routed, which would put the raw path in the span name and
// mint one name per path parameter -- the same cardinality mistake routePattern
// already exists to prevent for metric labels.
func tracing(tp trace.TracerProvider, propagator propagation.TextMapPropagator) func(http.Handler) http.Handler {
	tracer := tp.Tracer(tracerName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Adopt the caller's trace when there is one, so a trace spans
			// systems instead of restarting at this process.
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// The route is not known until routing has happened, so the span
			// opens under the method alone and is renamed below.
			ctx, span := tracer.Start(ctx, r.Method, trace.WithSpanKind(trace.SpanKindServer))
			defer span.End()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r.WithContext(ctx))

			status := ww.Status()
			if status == 0 {
				status = statusImplicitOK
			}

			route := routePattern(r)

			span.SetName(r.Method + " " + route)
			span.SetAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.HTTPRoute(route),
				semconv.HTTPResponseStatusCode(status),
				semconv.URLPath(r.URL.Path),
			)

			// Only server faults mark the span as an error. A 4xx is the
			// caller's mistake, and flagging those would bury real faults.
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}
		})
	}
}
