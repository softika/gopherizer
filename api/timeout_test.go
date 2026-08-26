package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

func TestRequestTimeoutCancelsTheContext(t *testing.T) {
	t.Parallel()

	var ctxErr error

	h := requestTimeout(config.HTTPConfig{RequestTimeout: 50 * time.Millisecond})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				ctxErr = r.Context().Err()
			case <-time.After(5 * time.Second):
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/slow", nil))

	require.Error(t, ctxErr, "a handler past the deadline must see a cancelled context")
	assert.ErrorIs(t, ctxErr, context.DeadlineExceeded)
}

func TestRequestTimeoutDisabledWhenUnset(t *testing.T) {
	t.Parallel()

	assert.Nil(t, requestTimeout(config.HTTPConfig{RequestTimeout: 0}))
	assert.Nil(t, requestTimeout(config.HTTPConfig{RequestTimeout: -time.Second}))
}

// TestTimeoutsAreLayered pins the ordering the whole design depends on: the
// innermost limit must fire first, so the error names the real cause instead of
// the outermost backstop tripping on everything.
func TestTimeoutsAreLayered(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)

	assert.Positive(t, cfg.Database.StatementTimeout)
	assert.Positive(t, cfg.Http.RequestTimeout)
	assert.Positive(t, cfg.Http.WriteTimeout)

	assert.Less(t, cfg.Database.StatementTimeout, cfg.Http.RequestTimeout,
		"a slow query must be cancelled by postgres before the request deadline")
	assert.Less(t, cfg.Http.RequestTimeout, cfg.Http.WriteTimeout,
		"the request deadline must fire before the write deadline backstop")
}
