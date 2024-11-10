package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/database/repositories/account"
	model "github.com/softika/gopherizer/internal/account"
)

func (s *RepositoriesTestSuite) TestAccountRepository_Create() {
	repo := account.NewRepository(s.dbService)

	tests := []struct {
		name    string
		input   *model.Account
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid account",
			input: &model.Account{
				Email:    "acc1@fake.com",
				Password: "password",
			},
			wantErr: assert.NoError,
		},
		{
			name:    "empty input",
			input:   &model.Account{},
			wantErr: assert.Error,
		},
		{
			name: "invalid email",
			input: &model.Account{
				Email:    "invalid-email",
				Password: "password",
			},
			wantErr: assert.Error,
		},
		{
			name: "existing email",
			input: &model.Account{
				Email: "john@mail.com",
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			acc, err := repo.Create(s.dbContainer.Ctx, tt.input)

			tt.wantErr(t, err, "Create() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(acc.Id)
			s.Assert().NotEmpty(acc.CreatedAt)
			s.Assert().NotEmpty(acc.UpdatedAt)
			s.Assert().Equal(tt.input.Email, acc.Email)
			s.Assert().Equal(tt.input.Password, acc.Password)
		})
	}
}

func (s *RepositoriesTestSuite) TestAccountRepository_GetByEmail() {
	repo := account.NewRepository(s.dbService)

	tests := []struct {
		name    string
		email   string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "valid email",
			email:   "john@mail.com",
			wantErr: assert.NoError,
		},
		{
			name:    "invalid email",
			email:   "invalid-email",
			wantErr: assert.Error,
		},
		{
			name:    "non-existent email",
			email:   "tester@fake.com",
			wantErr: assert.Error,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			identity, err := repo.GetByEmail(s.dbContainer.Ctx, tt.email)

			tt.wantErr(t, err, "GetByEmail() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil {
				return
			}

			s.Assert().NotEmpty(identity.AccountId)
			s.Assert().NotEmpty(identity.Email)
			s.Assert().NotEmpty(identity.Password)
			s.Assert().Equal(tt.email, identity.Email)
			s.Assert().NotEmpty(identity.Roles)
		})
	}
}

func (s *RepositoriesTestSuite) TestRepository_ChangePassword() {
	repo := account.NewRepository(s.dbService)

	type args struct {
		id          string
		oldPassword string
		newPassword string
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid password change",
			args: args{
				id:          "2f6f112a-a8e2-42c3-a6b0-c15e86d01704",
				oldPassword: "password",
				newPassword: "new-password",
			},
			wantErr: assert.NoError,
		},
		{
			name: "invalid old password",
			args: args{
				id:          "2f6f112a-a8e2-42c3-a6b0-c15e86d01704",
				oldPassword: "invalid-password",
				newPassword: "new-password",
			},
			wantErr: assert.Error,
		},
		{
			name: "invalid id",
			args: args{
				id:          "invalid-id",
				oldPassword: "password",
				newPassword: "new-password",
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			err := repo.ChangePassword(s.dbContainer.Ctx, tt.args.id, tt.args.oldPassword, tt.args.newPassword)

			tt.wantErr(t, err, "ChangePassword() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}
