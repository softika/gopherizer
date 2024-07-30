package testinfra

import (
	"context"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/config"
)

const (
	dbName     = "testdb"
	dbUser     = "test"
	dbPassword = "test"
	timeout    = time.Second * 240 // 4 minutes
)

type PostgresContainer struct {
	Ctx      context.Context
	Config   config.DatabaseConfig
	Shutdown func() error
}

func RunPostgres() (*PostgresContainer, error) {
	log := slogging.Slogger()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	req := testcontainers.ContainerRequest{
		Image:        "postgres:alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPassword,
			"POSTGRES_DB":       dbName,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(5 * time.Second),
	}

	dbContainer, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		cancel()
		return nil, err
	}
	var cfg config.DatabaseConfig
	for {
		if dbContainer.IsRunning() {
			log.Info("database container is running")
			host, err := dbContainer.Host(ctx)
			if err != nil {
				cancel()
				return nil, err
			}

			port, err := dbContainer.MappedPort(ctx, "5432/tcp")
			if err != nil {
				cancel()
				return nil, err
			}

			cfg = config.DatabaseConfig{
				Host:            host,
				Port:            port.Port(),
				DBName:          dbName,
				Password:        dbPassword,
				User:            dbUser,
				SSLModeDisabled: true,
			}
			break
		}
		log.Info("waiting for database container to start...")
	}

	return &PostgresContainer{
		Ctx:    ctx,
		Config: cfg,
		Shutdown: func() error {
			defer cancel()
			log.Info("terminating database container...")
			return dbContainer.Terminate(ctx)
		},
	}, nil
}
