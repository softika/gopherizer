package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

func TestBodyLimitAllowsBodiesUnderTheCap(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{MaxBodyBytes: 64}

	var got string
	h := bodyLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		got = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("small"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "small", got)
}

func TestBodyLimitRejectsOversizedBodies(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{MaxBodyBytes: 16}

	var readErr error
	h := bodyLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("a", 1024)))
	h.ServeHTTP(httptest.NewRecorder(), req)

	require.Error(t, readErr, "reading past the cap must fail")

	var maxErr *http.MaxBytesError
	assert.ErrorAs(t, readErr, &maxErr, "the failure must be the body cap, not something else")
}

// TestBodyLimitDisabledWhenUnset keeps the middleware honest about zero: the
// router omits it entirely rather than installing a cap of zero bytes, which
// would reject every request with a body.
func TestBodyLimitDisabledWhenUnset(t *testing.T) {
	t.Parallel()

	assert.Nil(t, bodyLimit(config.HTTPConfig{MaxBodyBytes: 0}))
	assert.Nil(t, bodyLimit(config.HTTPConfig{MaxBodyBytes: -1}))
}

// TestOversizedBodyIsRejectedByTheRealRouter exercises the whole stack: the cap
// has to be enforced before a handler can allocate the body.
func TestOversizedBodyIsRejectedByTheRealRouter(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)
	require.Positive(t, cfg.Http.MaxBodyBytes, "the shipped default must cap request bodies")

	router := NewRouter(cfg, stubDB{})

	huge := `{"firstName":"` + strings.Repeat("a", int(cfg.Http.MaxBodyBytes)+1024) + `","lastName":"x"}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profile", strings.NewReader(huge))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusCreated, w.Code, "an oversized body must not be accepted")
	assert.GreaterOrEqual(t, w.Code, 400)
	t.Logf("oversized body -> %d %s", w.Code, strings.TrimSpace(w.Body.String()))
}

// TestRaisedBodyLimitTakesEffect guards a trap: huma applies its own 1 MiB
// default to any operation that reads a body, so a configured limit above that
// would be silently overruled and the setting would appear not to work.
//
// A body over the huma default but under the configured one must fail on
// validation, never on size.
func TestRaisedBodyLimitTakesEffect(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)

	const humaDefault = 1024 * 1024
	cfg.Http.MaxBodyBytes = 4 * humaDefault

	router := NewRouter(cfg, stubDB{})

	// Comfortably over huma's default, comfortably under the configured cap.
	body := `{"firstName":"` + strings.Repeat("a", 2*humaDefault) + `","lastName":"x"}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusRequestEntityTooLarge, w.Code,
		"the configured limit must govern, not huma's default")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code,
		"the body should be read and then rejected by field validation")
}
