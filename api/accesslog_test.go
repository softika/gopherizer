package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/softika/slogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

// testLogger returns a logger writing JSON into a buffer, so records can be
// asserted on without touching the process-wide default.
func testLogger() (*slog.Logger, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	h := slogging.NewHandler(
		slogging.WithWriter(buf),
		slogging.WithLevel(slog.LevelDebug),
	)
	return slog.New(h), buf
}

func decodeRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "output: %s", buf.String())

	return got
}

func TestAccessLogRecordsRequest(t *testing.T) {
	t.Parallel()

	logger, buf := testLogger()

	r := chi.NewRouter()
	r.Use(accessLogger(config.HTTPConfig{}, logger))
	r.Get("/api/v1/profile/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hi"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/abc", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	got := decodeRecord(t, buf)
	assert.Equal(t, http.MethodGet, got["method"])
	assert.Equal(t, "/api/v1/profile/{id}", got["route"], "route must be the pattern, not the raw path")
	assert.EqualValues(t, http.StatusOK, got["status"])
	assert.EqualValues(t, 2, got["bytes"])
	assert.Contains(t, got, "duration_ms")
}

// TestAccessLogCarriesCorrelationIds is the point of replacing chi's logger:
// the busiest line in the system must be joinable to the application logs.
func TestAccessLogCarriesCorrelationIds(t *testing.T) {
	t.Parallel()

	logger, buf := testLogger()

	r := chi.NewRouter()
	r.Use(correlation)
	r.Use(accessLogger(config.HTTPConfig{}, logger))
	r.Get("/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.Header.Set(correlationIdHeader, "corr-abc")
	r.ServeHTTP(httptest.NewRecorder(), req)

	got := decodeRecord(t, buf)
	assert.Equal(t, "corr-abc", got[string(slogging.CorrelationIdKey)])
	assert.NotEmpty(t, got[string(slogging.RequestIdKey)])
}

func TestAccessLogLevelsByStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		want   string
	}{
		{"success is info", http.StatusOK, "INFO"},
		{"client error is warn", http.StatusNotFound, "WARN"},
		{"server error is error", http.StatusInternalServerError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger, buf := testLogger()

			h := accessLogger(config.HTTPConfig{}, logger)(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.status)
				}),
			)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

			got := decodeRecord(t, buf)
			assert.Equal(t, tt.want, got["level"])
		})
	}
}

// TestAccessLogSkipsMetricsPath keeps a 15s scrape interval from burying the
// logs it is meant to sit alongside.
func TestAccessLogSkipsMetricsPath(t *testing.T) {
	t.Parallel()

	logger, buf := testLogger()

	cfg := config.HTTPConfig{}
	cfg.Metrics.Path = "/metrics"

	h := accessLogger(cfg, logger)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/metrics", nil))

	assert.Zero(t, buf.Len(), "scrape requests must not be logged: %s", buf.String())
}

func TestAccessLogUsesContextForCancellation(t *testing.T) {
	t.Parallel()

	logger, buf := testLogger()

	h := accessLogger(config.HTTPConfig{}, logger)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	ctx := context.WithValue(context.Background(), slogging.RequestIdKey, "req-9")
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx)

	h.ServeHTTP(httptest.NewRecorder(), req)

	got := decodeRecord(t, buf)
	assert.Equal(t, "req-9", got[string(slogging.RequestIdKey)])
}
