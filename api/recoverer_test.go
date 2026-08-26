package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/softika/slogging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

func panicRouter(t *testing.T) (*chi.Mux, func() map[string]any) {
	t.Helper()

	logger, buf := testLogger()

	r := chi.NewRouter()
	r.Use(correlation)
	r.Use(recoverer(logger))
	r.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("something went badly wrong")
	})

	return r, func() map[string]any {
		t.Helper()
		var got map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &got), "log: %s", buf.String())
		return got
	}
}

func TestRecovererAnswers500(t *testing.T) {
	t.Parallel()

	r, _ := panicRouter(t)

	w := httptest.NewRecorder()
	require.NotPanics(t, func() {
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRecovererLogsThroughSlog is the whole point: chi's Recoverer writes the
// stack to stderr as plain text, so the most important line in the system was
// unparseable and had no correlation id.
func TestRecovererLogsThroughSlog(t *testing.T) {
	t.Parallel()

	r, record := panicRouter(t)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	got := record()
	assert.Equal(t, "ERROR", got["level"])
	assert.Contains(t, got["panic"], "something went badly wrong")
	assert.Contains(t, got["stack"], "api.")
	assert.Equal(t, "/boom", got["route"])
	assert.NotEmpty(t, got[string(slogging.CorrelationIdKey)], "a panic must be joinable to the request that caused it")
	assert.NotEmpty(t, got[string(slogging.RequestIdKey)])
}

// TestRecovererLeaksNothingToTheClient keeps the panic detail server-side. A
// stack trace names internal paths, types and sometimes values.
func TestRecovererLeaksNothingToTheClient(t *testing.T) {
	t.Parallel()

	r, _ := panicRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))

	body := w.Body.String()
	assert.NotContains(t, body, "something went badly wrong")
	assert.NotContains(t, body, "goroutine")
	assert.NotContains(t, strings.ToLower(body), "api.")
}

// TestRecovererMatchesHumaErrorShape keeps one error format on the wire: a
// client should not have to parse a panic differently from a validation error.
func TestRecovererMatchesHumaErrorShape(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)

	router := NewRouter(cfg, stubDB{})

	// A real huma error, for comparison.
	humaReq := httptest.NewRequest(http.MethodPost, "/api/v1/profile", strings.NewReader(`{"firstName":""}`))
	humaReq.Header.Set("Content-Type", "application/json")
	humaResp := httptest.NewRecorder()
	router.ServeHTTP(humaResp, humaReq)
	require.GreaterOrEqual(t, humaResp.Code, 400)

	logger, _ := testLogger()
	r := chi.NewRouter()
	r.Use(recoverer(logger))
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("x") })

	panicResp := httptest.NewRecorder()
	r.ServeHTTP(panicResp, httptest.NewRequest(http.MethodGet, "/boom", nil))

	assert.Equal(t, humaResp.Header().Get("Content-Type"), panicResp.Header().Get("Content-Type"),
		"a panic must use the same content type as every other error")

	var panicBody map[string]any
	require.NoError(t, json.Unmarshal(panicResp.Body.Bytes(), &panicBody))
	assert.Equal(t, "Internal Server Error", panicBody["title"])
	assert.EqualValues(t, 500, panicBody["status"])
}

// TestRecovererRepanicsAbortHandler preserves the escape hatch the standard
// library defines: ErrAbortHandler means "drop this connection deliberately",
// not "something broke".
func TestRecovererRepanicsAbortHandler(t *testing.T) {
	t.Parallel()

	logger, buf := testLogger()

	h := recoverer(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	assert.Panics(t, func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	})
	assert.Zero(t, buf.Len(), "a deliberate abort is not an error to report")
}
