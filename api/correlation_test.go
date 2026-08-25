package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/softika/slogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capture runs the middleware and reports what reached the handler.
func capture(t *testing.T, req *http.Request) (ctx context.Context, res *http.Response) {
	t.Helper()

	var got context.Context
	h := correlation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Context()
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	return got, w.Result()
}

func TestCorrelationGeneratesIds(t *testing.T) {
	t.Parallel()

	ctx, res := capture(t, httptest.NewRequest(http.MethodGet, "/", nil))
	defer func() { _ = res.Body.Close() }()

	requestId, ok := ctx.Value(slogging.RequestIdKey).(string)
	require.True(t, ok, "request id must be in the context for slogging to pick up")
	assert.NotEmpty(t, requestId)

	correlationId, ok := ctx.Value(slogging.CorrelationIdKey).(string)
	require.True(t, ok, "correlation id must be in the context")
	assert.NotEmpty(t, correlationId)

	// Callers need the ids back to correlate their own logs with ours.
	assert.Equal(t, requestId, res.Header.Get(requestIdHeader))
	assert.Equal(t, correlationId, res.Header.Get(correlationIdHeader))
}

// A correlation id spans systems, so an inbound one must be preserved.
func TestCorrelationPropagatesInboundCorrelationId(t *testing.T) {
	t.Parallel()

	const inbound = "11111111-2222-3333-4444-555555555555"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(correlationIdHeader, inbound)

	ctx, res := capture(t, req)
	defer func() { _ = res.Body.Close() }()

	assert.Equal(t, inbound, ctx.Value(slogging.CorrelationIdKey))
	assert.Equal(t, inbound, res.Header.Get(correlationIdHeader))
}

// Ids reach the logs, so a caller-supplied value must not be able to smuggle
// newlines or control characters into a log line.
func TestCorrelationRejectsUnsafeInboundIds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"newline injection", "abc\nlevel=ERROR msg=\"forged entry\""},
		{"carriage return", "abc\r\nfoo"},
		{"quotes", `abc"}{"level":"ERROR`},
		{"overlong", strings.Repeat("a", 512)},
		{"control characters", "abc\x00def"},
		{"spaces", "not a token"},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(correlationIdHeader, tt.value)
			req.Header.Set(requestIdHeader, tt.value)

			ctx, res := capture(t, req)
			defer func() { _ = res.Body.Close() }()

			correlationId, _ := ctx.Value(slogging.CorrelationIdKey).(string)
			requestId, _ := ctx.Value(slogging.RequestIdKey).(string)

			assert.NotEqual(t, tt.value, correlationId, "unsafe inbound id must be replaced")
			assert.NotEqual(t, tt.value, requestId, "unsafe inbound id must be replaced")
			assert.NotEmpty(t, correlationId, "a replacement id must still be issued")

			for _, bad := range []string{"\n", "\r", "\x00", `"`, " "} {
				assert.NotContains(t, correlationId, bad)
				assert.NotContains(t, requestId, bad)
			}
		})
	}
}

// Each request gets its own request id even when the correlation id is shared.
func TestCorrelationRequestIdIsPerRequest(t *testing.T) {
	t.Parallel()

	const shared = "11111111-2222-3333-4444-555555555555"

	newReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set(correlationIdHeader, shared)
		return r
	}

	ctx1, res1 := capture(t, newReq())
	defer func() { _ = res1.Body.Close() }()
	ctx2, res2 := capture(t, newReq())
	defer func() { _ = res2.Body.Close() }()

	assert.Equal(t, shared, ctx1.Value(slogging.CorrelationIdKey))
	assert.Equal(t, shared, ctx2.Value(slogging.CorrelationIdKey))
	assert.NotEqual(t, ctx1.Value(slogging.RequestIdKey), ctx2.Value(slogging.RequestIdKey),
		"each hop must get a distinct request id")
}
