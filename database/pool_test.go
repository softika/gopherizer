package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/testinfra"
)

func baseConfig(t *testing.T) *pgxpool.Config {
	t.Helper()

	c, err := pgxpool.ParseConfig("postgresql://u:p@localhost:5432/db?sslmode=disable")
	require.NoError(t, err)

	return c
}

func TestApplyPoolSettings(t *testing.T) {
	t.Parallel()

	poolCfg := baseConfig(t)

	applyPoolSettings(poolCfg, config.DatabaseConfig{
		MaxConns:          25,
		MinConns:          3,
		MaxConnLifetime:   time.Hour,
		MaxConnIdleTime:   30 * time.Minute,
		HealthCheckPeriod: time.Minute,
		StatementTimeout:  15 * time.Second,
	})

	assert.EqualValues(t, 25, poolCfg.MaxConns)
	assert.EqualValues(t, 3, poolCfg.MinConns)
	assert.Equal(t, time.Hour, poolCfg.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, poolCfg.MaxConnIdleTime)
	assert.Equal(t, time.Minute, poolCfg.HealthCheckPeriod)
	assert.Equal(t, "15000", poolCfg.ConnConfig.RuntimeParams["statement_timeout"],
		"postgres expects milliseconds")
}

// TestApplyPoolSettingsLeavesUnsetValuesAlone keeps zero meaning "not
// configured" rather than "zero connections", which pgx would reject.
func TestApplyPoolSettingsLeavesUnsetValuesAlone(t *testing.T) {
	t.Parallel()

	poolCfg := baseConfig(t)
	before := *poolCfg

	applyPoolSettings(poolCfg, config.DatabaseConfig{})

	assert.Equal(t, before.MaxConns, poolCfg.MaxConns)
	assert.Equal(t, before.MinConns, poolCfg.MinConns)
	assert.Equal(t, before.MaxConnLifetime, poolCfg.MaxConnLifetime)
	assert.NotContains(t, poolCfg.ConnConfig.RuntimeParams, "statement_timeout")
}

// TestShippedDefaultsPinPoolSize is the point of the setting: without it pgx
// derives the ceiling from the host CPU count, so the same image gets a
// different limit on every machine size.
func TestShippedDefaultsPinPoolSize(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)

	assert.Positive(t, cfg.Database.MaxConns, "the pool ceiling must be stated, not inherited from the host")
	assert.Positive(t, cfg.Database.StatementTimeout, "a runaway query must be bounded server-side")
	assert.Positive(t, cfg.Database.MaxConnLifetime)
}

// TestStatementTimeoutIsEnforcedByPostgres proves the setting reaches the
// server, rather than merely being present in the connection string.
func TestStatementTimeoutIsEnforcedByPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("requires docker")
	}

	container, err := testinfra.RunPostgres()
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Shutdown() })

	dbCfg := container.Config
	dbCfg.StatementTimeout = 300 * time.Millisecond

	svc, err := New(dbCfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Sleeps well past the timeout; Postgres must cancel it.
	var out int
	err = svc.Pool().QueryRow(ctx, "SELECT 1 FROM pg_sleep(5)").Scan(&out)

	require.Error(t, err, "a query past the statement timeout must be cancelled")
	assert.Contains(t, err.Error(), "canceling statement due to statement timeout")
}
