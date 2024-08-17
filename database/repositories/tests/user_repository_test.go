package tests

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"tldw/database/repositories/user"
	"tldw/internal/model"
)

func (s *RepositoriesTestSuite) TestUserRepository_Create() {
	repo := user.NewRepository(s.dbService)

	tests := []struct {
		name    string
		input   *model.User
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid user",
			input: &model.User{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john.doe@test.com",
				Password:  "password",
				Enabled:   true,
			},
			wantErr: assert.NoError,
		},
		{
			name:    "empty input",
			input:   &model.User{},
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
			s.Assert().Equal(tt.input.Password, u.Password)
			s.Assert().Equal(tt.input.Enabled, u.Enabled)

			u, err = repo.GetById(s.dbContainer.Ctx, u.Id)
			s.Assert().NoError(err)
			s.Assert().NotNil(u)
			s.Assert().NotEmpty(u.Id)
			s.Assert().NotEmpty(u.CreatedAt)
			s.Assert().NotEmpty(u.UpdatedAt)
			s.Assert().Equal(tt.input.FirstName, u.FirstName)
			s.Assert().Equal(tt.input.LastName, u.LastName)
			s.Assert().Equal(tt.input.Email, u.Email)
			s.Assert().Equal(tt.input.Password, u.Password)
			s.Assert().Equal(tt.input.Enabled, u.Enabled)

			if err = repo.DeleteById(s.dbContainer.Ctx, u.Id); err != nil {
				s.T().Errorf("failed to delete user: %v", err)
			}

			u, err = repo.GetById(s.dbContainer.Ctx, u.Id)
			s.Assert().Error(err)
			s.Assert().Nil(u)
		})
	}
}

func (s *RepositoriesTestSuite) TestUserRepository_Update() {
	repo := user.NewRepository(s.dbService)

	email := "john@email.com"
	u, err := repo.GetByEmail(s.dbContainer.Ctx, email)
	s.Assert().NoError(err)
	s.Assert().NotNil(u)

	tests := []struct {
		name    string
		input   *model.User
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid user",
			input: &model.User{
				Base: model.Base{
					Id: u.Id,
				},
				FirstName: "John",
				LastName:  "Johnson", //changed
				Email:     email,
				Password:  "password",
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			u, err = repo.Update(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "Update() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(u.Id)
			s.Assert().NotEmpty(u.CreatedAt)
			s.Assert().NotEmpty(u.UpdatedAt)
			s.Assert().Equal(tt.input.FirstName, u.FirstName)
			s.Assert().Equal(tt.input.LastName, u.LastName)
			s.Assert().Equal(tt.input.Email, u.Email)
			s.Assert().Equal(tt.input.Password, u.Password)
			s.Assert().Equal(tt.input.Enabled, u.Enabled)

			u, err = repo.GetByEmail(s.dbContainer.Ctx, u.Email)
			s.Assert().NoError(err)
			s.Assert().NotNil(u)
			s.Assert().NotEmpty(u.Id)
			s.Assert().NotEmpty(u.CreatedAt)
			s.Assert().NotEmpty(u.UpdatedAt)
			s.Assert().Equal(tt.input.FirstName, u.FirstName)
			s.Assert().Equal(tt.input.LastName, u.LastName)
			s.Assert().Equal(tt.input.Email, u.Email)
			s.Assert().Equal(tt.input.Password, u.Password)
			s.Assert().Equal(tt.input.Enabled, u.Enabled)
		})
	}
}
