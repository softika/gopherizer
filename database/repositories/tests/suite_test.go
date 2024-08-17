package tests

import (
	"testing"
	"tldw/database"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/suite"

	"tldw/logging"
	"tldw/testc"
)

type RepositoriesTestSuite struct {
	suite.Suite
	dbContainer *testc.PostgresContainer
	dbService   database.Service
}

func (s *RepositoriesTestSuite) SetupSuite() {
	var err error

	s.dbContainer, err = testc.RunPostgres()
	if err != nil {
		s.T().Fatal("failed to start postgres container", err)
	}

	s.dbService = database.New(s.dbContainer.Config)

	if err = goose.Up(s.dbService.DB(), "../../migrations"); err != nil {
		s.T().Fatal("failed to run migrations", err)
	}

	if err = goose.Up(s.dbService.DB(), "testdata"); err != nil {
		s.T().Fatal("failed to seed test data", err)
	}
}

func (s *RepositoriesTestSuite) TearDownSuite() {
	if err := s.dbService.Close(); err != nil {
		logging.Get().Warn("failed to close db connection", "error", err)
	}

	if err := s.dbContainer.Shutdown(); err != nil {
		logging.Get().Warn("failed to shutdown postgres container", "error", err)
	}
}

func TestRepositoriesTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping repository tests in short mode")
		return
	}
	suite.Run(t, new(RepositoriesTestSuite))
}
