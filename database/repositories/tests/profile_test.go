package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"tldw/database/repositories/profile"
	"tldw/internal/model"
)

func (s *RepositoriesTestSuite) TestProfileRepository_Create() {
	repo := profile.NewRepository(s.dbService)

	tests := []struct {
		name    string
		input   *model.Profile
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid profile",
			input: &model.Profile{
				AccountId: "2f6f112a-a8e2-42c3-a6b0-c15e86d01704",
				FirstName: "Milan",
				LastName:  "Miami",
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid account id",
			input: &model.Profile{
				AccountId: "invalid-account-id",
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: assert.Error,
		},
		{
			name:    "empty input",
			input:   &model.Profile{},
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
	repo := profile.NewRepository(s.dbService)

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
