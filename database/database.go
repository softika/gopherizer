package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"tldw/config"
	"tldw/logger"
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
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// DB returns the database connection.
	DB() *sql.DB

	// Pool returns the pgx connection pool.
	Pool() *pgxpool.Pool
}

type service struct {
	db   *sql.DB
	pool *pgxpool.Pool
}

var (
	dbService *service

	once sync.Once
)

func New(cfg config.DatabaseConfig) Service {
	once.Do(func() {
		log := logger.Get()
		log.Info("creating a new database connection pool...")

		pool, err := pgxpool.New(context.Background(), dsnFromConfig(cfg))
		if err != nil {
			log.Error("failed to create db connection pool", "error", err)
			panic(err)
		}

		db := stdlib.OpenDBFromPool(pool)
		if err = db.Ping(); err != nil {
			log.Error("failed to ping db", "error", err)
			panic(err)
		}

		dbService = &service{
			db:   db,
			pool: pool,
		}
	})

	return dbService
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	log := logger.Get()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Error("db is down", "error", err)
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	logger.Get().Info("closing the database connection...")
	return s.db.Close()
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
	return s.db
}

func (s *service) Pool() *pgxpool.Pool {
	return s.pool
}
