package health_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/internal/health"
	"github.com/softika/gopherizer/pkg/errorx"
)

type fakeRepo struct {
	stats map[string]string
}

func (f fakeRepo) Health(context.Context) map[string]string { return f.stats }

// Pool internals describe infrastructure, not service health. They belong in
// server-side logs, never in a response an unauthenticated caller can read.
func TestCheckDoesNotExposeInternals(t *testing.T) {
	t.Parallel()

	repo := fakeRepo{stats: map[string]string{
		"status":               "up",
		"message":              "It's healthy",
		"max_connections":      "50",
		"acquired_connections": "7",
		"acquire_duration":     "1.2ms",
	}}

	res, err := health.NewService(repo).Check(context.Background(), health.Request{})
	require.NoError(t, err)
	require.NotNil(t, res)

	body, err := json.Marshal(res)
	require.NoError(t, err)

	for _, leak := range []string{
		"max_connections", "acquired_connections", "acquire_duration", "50", "7",
	} {
		assert.NotContainsf(t, string(body), leak,
			"health response leaked %q: %s", leak, body)
	}

	assert.Contains(t, string(body), "up")
}

// A failing dependency must surface as unavailable so orchestrators stop
// routing traffic, rather than reporting a healthy 200.
func TestCheckReportsUnavailable(t *testing.T) {
	t.Parallel()

	repo := fakeRepo{stats: map[string]string{
		"status": "down",
		"error":  "db down: connection refused host=10.0.3.7",
	}}

	res, err := health.NewService(repo).Check(context.Background(), health.Request{})

	require.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, errorx.ErrUnavailable, errorx.TypeOf(err))
	assert.NotContains(t, err.Error(), "10.0.3.7",
		"the client-facing error must not carry infrastructure detail")
}

// Liveness answers "is the process running", so it must never touch a
// dependency; otherwise a database blip restarts healthy pods.
func TestLiveDoesNotTouchDependencies(t *testing.T) {
	t.Parallel()

	repo := fakeRepo{stats: map[string]string{"status": "down"}}

	res, err := health.NewService(repo).Live(context.Background(), health.Request{})

	require.NoError(t, err, "liveness must succeed even when the database is down")
	require.NotNil(t, res)
	assert.Equal(t, "up", res.Status)
}
