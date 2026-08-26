package database

import (
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/softika/gopherizer/config"
)

// statementTimeoutParam is the Postgres runtime setting that bounds a query
// server-side.
const statementTimeoutParam = "statement_timeout"

// applyPoolSettings copies tuning from configuration onto the pool.
//
// Every value is optional, and zero means "not configured" rather than zero --
// pgx rejects a pool of zero connections, and an unset duration should keep
// pgx's own default rather than disable the behaviour outright.
//
// Stating these matters more than the specific numbers. Left unset, pgx derives
// MaxConns from the host CPU count, so the same image quietly gets a different
// ceiling on every machine size -- while the database's connection limit is a
// shared budget that no single service should size by accident.
func applyPoolSettings(poolCfg *pgxpool.Config, cfg config.DatabaseConfig) {
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}

	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}

	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}

	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	if cfg.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	}

	if cfg.StatementTimeout > 0 {
		if poolCfg.ConnConfig.RuntimeParams == nil {
			poolCfg.ConnConfig.RuntimeParams = make(map[string]string)
		}

		// Postgres expects milliseconds. Enforcing it server-side is what stops
		// a runaway query holding a pooled connection until the HTTP write
		// deadline: cancelling the client's context alone would not.
		poolCfg.ConnConfig.RuntimeParams[statementTimeoutParam] =
			strconv.FormatInt(cfg.StatementTimeout.Milliseconds(), 10)
	}
}
