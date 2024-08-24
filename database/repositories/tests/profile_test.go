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
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john.doe@test.com",
			},
			wantErr: assert.NoError,
		},
		{
			name:    "empty input",
			input:   &model.Profile{},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			u, err := repo.Create(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "Create() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(u.Id)
			s.Assert().NotEmpty(u.CreatedAt)
			s.Assert().NotEmpty(u.UpdatedAt)
			s.Assert().Equal(tt.input.FirstName, u.FirstName)
			s.Assert().Equal(tt.input.LastName, u.LastName)
			s.Assert().Equal(tt.input.Email, u.Email)

			u, err = repo.GetById(s.dbContainer.Ctx, u.Id)
			s.Assert().NoError(err)
			s.Assert().NotNil(u)
		})
	}
}
