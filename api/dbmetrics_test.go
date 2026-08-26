package api

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/config"
)

// nilPoolDB models a service whose pool is not available.
type nilPoolDB struct{}

func (nilPoolDB) Health(context.Context) map[string]string { return nil }
func (nilPoolDB) Close() error                             { return nil }
func (nilPoolDB) DB() *sql.DB                              { return nil }
func (nilPoolDB) Pool() *pgxpool.Pool                      { return nil }

func fixedStats() poolStats {
	return poolStats{
		MaxConns:                10,
		AcquiredConns:           3,
		IdleConns:               2,
		ConstructingConns:       1,
		AcquireCount:            100,
		EmptyAcquireCount:       7,
		CanceledAcquireCount:    2,
		AcquireDuration:         1500 * time.Millisecond,
		MaxIdleDestroyCount:     4,
		MaxLifetimeDestroyCount: 5,
	}
}

func TestPoolMetricsExportConnectionStates(t *testing.T) {
	t.Parallel()

	m := newMetrics(withPoolStats(fixedStats))
	body := scrape(t, m.handler())

	assert.Contains(t, body, `db_pool_connections{state="acquired"} 3`)
	assert.Contains(t, body, `db_pool_connections{state="idle"} 2`)
	assert.Contains(t, body, `db_pool_connections{state="constructing"} 1`)
	assert.Contains(t, body, `db_pool_connections_max 10`)
}

// TestPoolMetricsExportSaturationSignals covers the numbers that actually page
// someone: waiting for a connection means the pool is too small.
func TestPoolMetricsExportSaturationSignals(t *testing.T) {
	t.Parallel()

	m := newMetrics(withPoolStats(fixedStats))
	body := scrape(t, m.handler())

	assert.Contains(t, body, `db_pool_acquires_total 100`)
	assert.Contains(t, body, `db_pool_acquire_waits_total 7`)
	assert.Contains(t, body, `db_pool_acquire_cancels_total 2`)
	assert.Contains(t, body, `db_pool_acquire_duration_seconds_total 1.5`)
}

func TestPoolMetricsExportCloseReasons(t *testing.T) {
	t.Parallel()

	m := newMetrics(withPoolStats(fixedStats))
	body := scrape(t, m.handler())

	assert.Contains(t, body, `db_pool_connections_closed_total{reason="idle"} 4`)
	assert.Contains(t, body, `db_pool_connections_closed_total{reason="lifetime"} 5`)
}

// TestPoolMetricsReadAtScrapeTime proves the collector has no cached state, so
// a scrape always reflects the pool as it is now.
func TestPoolMetricsReadAtScrapeTime(t *testing.T) {
	t.Parallel()

	acquired := int32(1)
	m := newMetrics(withPoolStats(func() poolStats {
		return poolStats{AcquiredConns: acquired}
	}))

	assert.Contains(t, scrape(t, m.handler()), `db_pool_connections{state="acquired"} 1`)

	acquired = 8

	assert.Contains(t, scrape(t, m.handler()), `db_pool_connections{state="acquired"} 8`)
}

func TestBuildInfoExported(t *testing.T) {
	t.Parallel()

	cfg := config.AppConfig{Name: "gopherizer", Version: "1.2.3", Environment: "staging"}

	m := newMetrics(withBuildInfo(cfg))
	body := scrape(t, m.handler())

	assert.Contains(t, body, "app_build_info")
	assert.Contains(t, body, `name="gopherizer"`)
	assert.Contains(t, body, `version="1.2.3"`)
	assert.Contains(t, body, `environment="staging"`)
	assert.Contains(t, body, `go_version="`)
}

// TestMetricsWithoutOptionsOmitsExtras keeps the default construction cheap:
// a router with no database still builds a working registry.
func TestMetricsWithoutOptionsOmitsExtras(t *testing.T) {
	t.Parallel()

	body := scrape(t, newMetrics().handler())

	assert.NotContains(t, body, "db_pool_connections")
	assert.NotContains(t, body, "app_build_info")
}

// TestPoolStatsFromNilPoolDoesNotPanic guards the scrape path: collection runs
// inside a Prometheus scrape, and a panic there would take down the endpoint
// that is meant to explain what is wrong.
func TestPoolStatsFromNilPoolDoesNotPanic(t *testing.T) {
	t.Parallel()

	stats := poolStatsFrom(nilPoolDB{})

	assert.NotPanics(t, func() {
		assert.Equal(t, poolStats{}, stats())
	})
}
