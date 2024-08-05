package tests

import (
	"github.com/stretchr/testify/suite"
	"testing"
	"tldw/database"
	"tldw/http/api"
	"tldw/logger"
	"tldw/testc"
)

type E2ETestSuite struct {
	suite.Suite
	dbContainer *testc.PostgresContainer
	dbService   database.Service
	router      *api.Router
}

func (s *E2ETestSuite) SetupSuite() {
	var err error

	s.dbContainer, err = testc.RunPostgres()
	if err != nil {
		s.T().Fatal("failed to start postgres container", err)
	}

	s.dbService = database.New(s.dbContainer.Config)
	s.router = api.NewRouter("test", "test-secret-key")
}

func (s *E2ETestSuite) TearDownSuite() {
	if err := s.dbService.Close(); err != nil {
		logger.Get().Warn("failed to close db connection", "error", err)
	}

	if err := s.dbContainer.Shutdown(); err != nil {
		logger.Get().Warn("failed to shutdown postgres container", "error", err)
	}
}

func TestE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e tests in short mode")
		return
	}
	suite.Run(t, new(E2ETestSuite))
}
