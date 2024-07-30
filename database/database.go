package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/config"
)

//go:embed migrations/*.sql
var migrations embed.FS

func GetMigrationFS() embed.FS {
	return migrations
}

func GetDialect() string {
	return "postgres"
}

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health(ctx context.Context) map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// DB returns the database connection.
	DB() *sql.DB

	// Pool returns the pgx connection pool.
	Pool() *pgxpool.Pool
}

type service struct {
	pool *pgxpool.Pool
}

var (
	dbService *service

	once sync.Once
)

func New(cfg config.DatabaseConfig) Service {
	once.Do(func() {
		logger := slogging.Slogger()
		logger.Info("creating a new database connection pool...")

		ctx := context.Background()

		pool, err := pgxpool.New(ctx, dsnFromConfig(cfg))
		if err != nil {
			logger.Error("failed to create db connection pool", "error", err)
			panic(err)
		}

		if err = pool.Ping(ctx); err != nil {
			logger.Error("failed to ping db", "error", err)
			panic(err)
		}

		dbService = &service{
			pool: pool,
		}
	})

	return dbService
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health(ctx context.Context) map[string]string {
	log := slogging.Slogger()

	stats := make(map[string]string)

	// Ping the database
	err := s.pool.Ping(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.ErrorContext(ctx, "db is down", "error", err)
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Logger database stats (like open connections, in use, idle, etc.)
	poolStat := s.pool.Stat()
	stats["max_connections"] = strconv.Itoa(int(poolStat.MaxConns()))
	stats["total_connections"] = strconv.Itoa(int(poolStat.TotalConns()))
	stats["acquired_connections"] = strconv.Itoa(int(poolStat.AcquiredConns()))
	stats["new_acquired_connections"] = strconv.FormatInt(poolStat.NewConnsCount(), 10)
	stats["empty_acquire_count"] = strconv.FormatInt(poolStat.EmptyAcquireCount(), 10)
	stats["canceled_acquire_count"] = strconv.FormatInt(poolStat.CanceledAcquireCount(), 10)
	stats["acquire_count"] = strconv.FormatInt(poolStat.AcquireCount(), 10)
	stats["acquire_duration"] = poolStat.AcquireDuration().String()
	stats["idle_connections"] = strconv.Itoa(int(poolStat.IdleConns()))
	stats["constructing_connections"] = strconv.Itoa(int(poolStat.ConstructingConns()))
	stats["max_idle_destroy_count"] = strconv.FormatInt(poolStat.MaxIdleDestroyCount(), 10)
	stats["max_lifetime_destroy_count"] = strconv.FormatInt(poolStat.MaxLifetimeDestroyCount(), 10)

	// Evaluate stats to provide a health message
	if poolStat.TotalConns() > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if poolStat.ConstructingConns() > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if poolStat.MaxIdleDestroyCount() > int64(poolStat.TotalConns())/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if poolStat.MaxLifetimeDestroyCount() > int64(poolStat.TotalConns())/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	slogging.Slogger().Info("closing the database connection...")
	s.pool.Close()
	return nil
}

func dsnFromConfig(config config.DatabaseConfig) string {
	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=require",
		config.User, config.Password, config.Host, config.Port, config.DBName,
	)

	if config.SSLModeDisabled {
		dsn = strings.Replace(dsn, "sslmode=require", "sslmode=disable", 1)
	}

	return dsn
}

func (s *service) DB() *sql.DB {
	return stdlib.OpenDBFromPool(s.pool)
}

func (s *service) Pool() *pgxpool.Pool {
	return s.pool
}
