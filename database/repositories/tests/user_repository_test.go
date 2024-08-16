package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

			assert.NotEmpty(t, u.Id)
			assert.NotEmpty(t, u.CreatedAt)
			assert.NotEmpty(t, u.UpdatedAt)
			assert.Equal(t, tt.input.FirstName, u.FirstName)
			assert.Equal(t, tt.input.LastName, u.LastName)
			assert.Equal(t, tt.input.Email, u.Email)
			assert.Equal(t, tt.input.Password, u.Password)
			assert.Equal(t, tt.input.Enabled, u.Enabled)
		})
	}

}
