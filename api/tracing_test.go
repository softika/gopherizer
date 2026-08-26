package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/softika/slogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/otelx"
)

// tracedRouter mounts a parameterised route behind the tracing middleware and
// returns the exporter holding the finished spans.
func tracedRouter(t *testing.T) (*chi.Mux, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	r := chi.NewRouter()
	r.Use(tracing(tp, propagation.TraceContext{}))
	r.Get("/api/v1/profile/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	return r, exporter
}

func TestTracingCreatesServerSpan(t *testing.T) {
	t.Parallel()

	r, exporter := tracedRouter(t)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/profile/abc", nil))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "GET /api/v1/profile/{id}", span.Name, "span name must use the route pattern")
	assert.Equal(t, trace.SpanKindServer, span.SpanKind)

	attrs := make(map[string]string)
	for _, a := range span.Attributes {
		attrs[string(a.Key)] = a.Value.String()
	}
	assert.Equal(t, "/api/v1/profile/{id}", attrs["http.route"])
	assert.Equal(t, "GET", attrs["http.request.method"])
	assert.Equal(t, "200", attrs["http.response.status_code"])
}

// TestTracingSpanNameUsesPatternNotRawPath keeps span names bounded. Tempo
// indexes by name, so one name per profile id would be the same cardinality
// mistake the metrics labels already avoid.
func TestTracingSpanNameUsesPatternNotRawPath(t *testing.T) {
	t.Parallel()

	r, exporter := tracedRouter(t)

	for _, id := range []string{"aaaa", "bbbb", "cccc"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/profile/"+id, nil))
	}

	names := make(map[string]int)
	for _, s := range exporter.GetSpans() {
		names[s.Name]++
	}

	assert.Len(t, names, 1, "all three requests must share one span name, got %v", names)
	assert.Equal(t, 3, names["GET /api/v1/profile/{id}"])
}

// TestTracingContinuesUpstreamTrace is what makes a trace span systems rather
// than stopping at this process.
func TestTracingContinuesUpstreamTrace(t *testing.T) {
	t.Parallel()

	r, exporter := tracedRouter(t)

	const upstreamTraceId = "4bf92f3577b34da6a3ce929d0e0e4736"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/abc", nil)
	req.Header.Set("traceparent", "00-"+upstreamTraceId+"-00f067aa0ba902b7-01")

	r.ServeHTTP(httptest.NewRecorder(), req)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	assert.Equal(t, upstreamTraceId, spans[0].SpanContext.TraceID().String())
	assert.Equal(t, "00f067aa0ba902b7", spans[0].Parent.SpanID().String())
}

func TestTracingMarksServerErrors(t *testing.T) {
	t.Parallel()

	r, exporter := tracedRouter(t)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	assert.Equal(t, codes.Error, spans[0].Status.Code)
}

// TestTracingLeavesClientErrorsUnset follows the OpenTelemetry convention: a
// 4xx is the caller's fault, not a failure of this server, and marking it an
// error would drown real faults in noise.
func TestTracingLeavesClientErrorsUnset(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSampler(sdktrace.AlwaysSample()))
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	r := chi.NewRouter()
	r.Use(tracing(tp, propagation.TraceContext{}))
	r.Get("/nope", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nope", nil))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status.Code)
}

// TestTracingReachesTheAccessLog is the end-to-end proof of the design: a
// request produces a span, the span reaches the log record, and the record also
// keeps its correlation ids. That is what makes a Grafana exemplar clickable
// through to a trace and onward to the logs for the same request.
func TestTracingReachesTheAccessLog(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	buf := new(bytes.Buffer)
	logger := slog.New(slogging.NewHandler(
		slogging.WithWriter(buf),
		slogging.WithLevel(slog.LevelDebug),
		slogging.WithExtractor(slogging.ContextIds, otelx.TraceAttrs),
	))

	r := chi.NewRouter()
	r.Use(tracing(tp, propagation.TraceContext{}))
	r.Use(correlation)
	r.Use(accessLogger(config.HTTPConfig{}, logger))
	r.Get("/ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok", nil))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "log output: %s", buf.String())

	assert.Equal(t, spans[0].SpanContext.TraceID().String(), got["trace_id"],
		"the access log must carry the trace id of the request it describes")
	assert.NotEmpty(t, got[string(slogging.CorrelationIdKey)],
		"correlation ids must survive alongside trace ids, not be replaced by them")
}
