package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

// A constructor must report failure to its caller instead of panicking, so the
// server can log a clear error and exit rather than dumping a stack trace.
func TestNewReturnsErrorForUnreachableDatabase(t *testing.T) {
	t.Parallel()

	cfg := config.DatabaseConfig{
		Host:            "127.0.0.1",
		Port:            "1", // nothing listens here
		DBName:          "nope",
		User:            "nope",
		Password:        "nope",
		SSLModeDisabled: true,
	}

	var (
		svc Service
		err error
	)

	require.NotPanics(t, func() {
		svc, err = New(cfg)
	}, "an unreachable database must not panic the process")

	require.Error(t, err)
	assert.Nil(t, svc)
}

// Each call must yield an independent pool. A process-wide singleton cannot be
// reopened once closed, and silently ignores a second, different config.
func TestNewReturnsIndependentPools(t *testing.T) {
	first, err := New(dbCfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Close() })

	second, err := New(dbCfg)
	require.NoError(t, err)

	assert.NotSame(t, first, second, "New must not hand back a shared instance")

	// Closing one must leave the other usable.
	require.NoError(t, second.Close())
	assert.Equal(t, "up", first.Health(context.Background())["status"])
}
