package api

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
)

// poolStats is the subset of connection pool statistics worth exporting.
//
// It exists so the collector does not depend on pgx: *pgxpool.Stat has no
// exported fields and cannot be constructed, which would force every test of
// this code to stand up a real database.
type poolStats struct {
	MaxConns          int32
	AcquiredConns     int32
	IdleConns         int32
	ConstructingConns int32

	AcquireCount      int64
	EmptyAcquireCount int64
	// CanceledAcquireCount counts acquires abandoned before a connection freed.
	CanceledAcquireCount int64
	// AcquireDuration is the total duration of all successful acquires, which
	// includes constructing new connections -- it is not queueing time alone.
	// EmptyAcquireCount is the signal for a pool that is actually too small.
	AcquireDuration time.Duration

	MaxIdleDestroyCount     int64
	MaxLifetimeDestroyCount int64
}

// poolStatsFrom reads live statistics from the pool.
//
// A nil pool yields zero statistics rather than panicking. Collection runs
// inside a scrape, and a scrape must never be able to bring the process down --
// least of all while the database is the thing going wrong.
func poolStatsFrom(db database.Service) func() poolStats {
	return func() poolStats {
		pool := db.Pool()
		if pool == nil {
			return poolStats{}
		}

		s := pool.Stat()

		return poolStats{
			MaxConns:                s.MaxConns(),
			AcquiredConns:           s.AcquiredConns(),
			IdleConns:               s.IdleConns(),
			ConstructingConns:       s.ConstructingConns(),
			AcquireCount:            s.AcquireCount(),
			EmptyAcquireCount:       s.EmptyAcquireCount(),
			CanceledAcquireCount:    s.CanceledAcquireCount(),
			AcquireDuration:         s.AcquireDuration(),
			MaxIdleDestroyCount:     s.MaxIdleDestroyCount(),
			MaxLifetimeDestroyCount: s.MaxLifetimeDestroyCount(),
		}
	}
}

// poolCollector exports connection pool statistics.
//
// These numbers were already being computed for the readiness probe and then
// discarded into a map of strings. Pool saturation is what actually pages
// someone, and it was the one signal missing from Prometheus.
//
// It reads at scrape time and holds no state of its own, so a scrape always
// reflects the pool as it is now rather than as it was when something last
// bothered to sample it.
type poolCollector struct {
	stats func() poolStats

	connections    *prometheus.Desc
	maxConnections *prometheus.Desc
	acquires       *prometheus.Desc
	acquireWaits   *prometheus.Desc
	acquireCancels *prometheus.Desc
	acquireSeconds *prometheus.Desc
	closed         *prometheus.Desc
}

func newPoolCollector(stats func() poolStats) *poolCollector {
	return &poolCollector{
		stats: stats,
		connections: prometheus.NewDesc(
			"db_pool_connections",
			"Connections in the pool, by state.",
			[]string{"state"}, nil,
		),
		maxConnections: prometheus.NewDesc(
			"db_pool_connections_max",
			"Maximum size of the connection pool.",
			nil, nil,
		),
		acquires: prometheus.NewDesc(
			"db_pool_acquires_total",
			"Total number of connection acquisitions.",
			nil, nil,
		),
		acquireWaits: prometheus.NewDesc(
			"db_pool_acquire_waits_total",
			"Acquisitions that had to wait because the pool was empty.",
			nil, nil,
		),
		acquireCancels: prometheus.NewDesc(
			"db_pool_acquire_cancels_total",
			"Acquisitions cancelled before a connection became available.",
			nil, nil,
		),
		acquireSeconds: prometheus.NewDesc(
			"db_pool_acquire_duration_seconds_total",
			"Total duration of successful acquires, including time spent constructing new connections.",
			nil, nil,
		),
		closed: prometheus.NewDesc(
			"db_pool_connections_closed_total",
			"Connections closed by the pool, by reason.",
			[]string{"reason"}, nil,
		),
	}
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.connections
	ch <- c.maxConnections
	ch <- c.acquires
	ch <- c.acquireWaits
	ch <- c.acquireCancels
	ch <- c.acquireSeconds
	ch <- c.closed
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.stats()

	gauge := func(d *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
	}
	counter := func(d *prometheus.Desc, v float64, labels ...string) {
		ch <- prometheus.MustNewConstMetric(d, prometheus.CounterValue, v, labels...)
	}

	gauge(c.connections, float64(s.AcquiredConns), "acquired")
	gauge(c.connections, float64(s.IdleConns), "idle")
	gauge(c.connections, float64(s.ConstructingConns), "constructing")
	gauge(c.maxConnections, float64(s.MaxConns))

	counter(c.acquires, float64(s.AcquireCount))
	counter(c.acquireWaits, float64(s.EmptyAcquireCount))
	counter(c.acquireCancels, float64(s.CanceledAcquireCount))
	counter(c.acquireSeconds, s.AcquireDuration.Seconds())

	counter(c.closed, float64(s.MaxIdleDestroyCount), "idle")
	counter(c.closed, float64(s.MaxLifetimeDestroyCount), "lifetime")
}

// withPoolStats exports connection pool statistics.
func withPoolStats(stats func() poolStats) metricsOption {
	return func(m *metrics) {
		m.registry.MustRegister(newPoolCollector(stats))
	}
}

// withBuildInfo exports a constant series identifying the running build.
//
// The value carries no information; the labels do. Joining against it is what
// lets a dashboard line a latency change up against the deploy that caused it.
func withBuildInfo(cfg config.AppConfig) metricsOption {
	return func(m *metrics) {
		info := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "app_build_info",
			Help: "Build and deployment identity of the running process.",
		}, []string{"name", "version", "environment", "go_version"})

		info.WithLabelValues(cfg.Name, cfg.Version, cfg.Environment, runtime.Version()).Set(1)

		m.registry.MustRegister(info)
	}
}
