package tests

import (
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/api"
	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
	"github.com/softika/gopherizer/pkg/testinfra"
)

type E2ETestSuite struct {
	suite.Suite

	dbContainer *testinfra.PostgresContainer
	dbService   database.Service
	router      *api.Router
}

func (s *E2ETestSuite) SetupSuite() {
	var err error

	s.dbContainer, err = testinfra.RunPostgres()
	if err != nil {
		s.T().Fatal("failed to start postgres container", err)
	}

	s.dbService = database.New(s.dbContainer.Config)

	s.prepareDb()

	cfg := &config.Config{
		App: config.AppConfig{Environment: "test"},
		Auth: config.AuthConfig{
			Secret:   "test-secret-key",
			TokenExp: 20,
		},
		Database: s.dbContainer.Config,
	}
	s.router = api.NewRouter(cfg)
}

func (s *E2ETestSuite) prepareDb() {
	if err := goose.Up(s.dbService.DB(), "../database/migrations"); err != nil {
		s.T().Fatal("failed to run migrations", err)
	}

	if err := goose.Up(s.dbService.DB(), "testdata"); err != nil {
		s.T().Fatal("failed to seed test data", err)
	}
}

func (s *E2ETestSuite) TearDownSuite() {
	if err := s.dbService.Close(); err != nil {
		slogging.Slogger().Warn("failed to close db connection", "error", err)
	}

	if err := s.dbContainer.Shutdown(); err != nil {
		slogging.Slogger().Warn("failed to shutdown postgres container", "error", err)
	}
}

func TestE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
		return
	}
	suite.Run(t, new(E2ETestSuite))
}
