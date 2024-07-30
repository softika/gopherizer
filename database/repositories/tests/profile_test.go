package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/database/repositories"
	"github.com/softika/gopherizer/internal/profile"
)

func (s *RepositoriesTestSuite) TestProfileRepository_Create() {
	repo := repositories.NewProfileRepository(s.dbService)

	tests := []struct {
		name    string
		input   *profile.Profile
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid profile",
			input: profile.New().
				WithFirstName("Milan").
				WithLastName("Miami"),
			wantErr: assert.NoError,
		},
		{
			name:    "empty input",
			input:   &profile.Profile{},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			p, err := repo.Create(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "Create() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(p.Id)
			s.Assert().NotEmpty(p.CreatedAt)
			s.Assert().NotEmpty(p.UpdatedAt)
			s.Assert().Equal(tt.input.FirstName, p.FirstName)
			s.Assert().Equal(tt.input.LastName, p.LastName)
		})
	}
}

func (s *RepositoriesTestSuite) TestProfileRepository_GetById() {
	repo := repositories.NewProfileRepository(s.dbService)

	tests := []struct {
		name    string
		input   string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "empty id",
			input:   "",
			wantErr: assert.Error,
		},
		{
			name:    "invalid id",
			input:   "invalid-id",
			wantErr: assert.Error,
		},
		{
			name:    "valid id",
			input:   "0dd35f9a-0d20-41f1-80c2-d7993e313fb4",
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			p, err := repo.GetById(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "GetById() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(p.Id)
			s.Assert().NotEmpty(p.CreatedAt)
			s.Assert().NotEmpty(p.UpdatedAt)
		})
	}
}

func (s *RepositoriesTestSuite) TestProfileRepository_Update() {
	repo := repositories.NewProfileRepository(s.dbService)

	tests := []struct {
		name    string
		input   *profile.Profile
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid profile",
			input: profile.New().
				WithId("0dd35f9a-0d20-41f1-80c2-d7993e313fb4").
				WithFirstName("Lanmi").
				WithLastName("Miami"),
			wantErr: assert.NoError,
		},
		{
			name: "invalid id",
			input: profile.New().
				WithId("invalid-id").
				WithFirstName("John").
				WithLastName("Doe"),
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			p, err := repo.Update(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "Update() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(p.Id)
			s.Assert().NotEmpty(p.CreatedAt)
			s.Assert().NotEmpty(p.UpdatedAt)
			s.Assert().Equal(tt.input.FirstName, p.FirstName)
			s.Assert().Equal(tt.input.LastName, p.LastName)
		})
	}
}

func (s *RepositoriesTestSuite) TestProfileRepository_DeleteById() {
	repo := repositories.NewProfileRepository(s.dbService)

	tests := []struct {
		name    string
		input   string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "empty id",
			input:   "",
			wantErr: assert.Error,
		},
		{
			name:    "invalid id",
			input:   "invalid-id",
			wantErr: assert.Error,
		},
		{
			name:    "valid id",
			input:   "0dd35f9a-0d20-41f1-80c2-d7993e313fb6",
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			err := repo.DeleteById(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "DeleteById() error = %v, wantErr %v", err, tt.wantErr)

			if err != nil {
				got, err := repo.GetById(s.dbContainer.Ctx, tt.input)
				s.Assert().Error(err)
				s.Assert().Nil(got)
			}
		})
	}
}
