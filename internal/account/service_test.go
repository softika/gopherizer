package account_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/internal/account"
	"github.com/softika/gopherizer/internal/account/mock"
)

func TestService_Login(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.AuthConfig{
		Secret:   "secret",
		TokenExp: 3,
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     account.LoginRequest
		mockFn  func(m *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			req: account.LoginRequest{
				Email:    "user@fake.com",
				Password: "password",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().GetByEmail(ctx, "user@fake.com").Return(&account.Identity{
					Email:    "user@fake.com",
					Password: "$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq",
				}, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid credentials",
			req: account.LoginRequest{
				Email:    "user@fake.com",
				Password: "invalid",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().GetByEmail(ctx, "user@fake.com").Return(&account.Identity{
					Email:    "user@fake.com",
					Password: "$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq",
				}, nil)
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid Email",
			req: account.LoginRequest{
				Email:    "invalid",
				Password: "password",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().GetByEmail(ctx, "invalid").Return(nil, assert.AnError)
			},
			wantErr: assert.Error,
		},
	}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mock.NewMockRepository(ctrl)
			s := account.NewService(cfg, repo)

			tt.mockFn(repo)

			res, err := s.Login(ctx, tt.req)
			if err != nil && tt.wantErr(t, err) {
				return
			}

			assert.NotNil(t, res)
			assert.NotNil(t, res.Token)
		})
	}
}

func TestService_Register(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.AuthConfig{
		Secret:   "secret",
		TokenExp: 3,
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     account.RegisterRequest
		mockFn  func(m *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			req: account.RegisterRequest{
				Email:    "user@fake.com",
				Password: "password",
			},
			mockFn: func(m *mock.MockRepository) {
				got := account.New().
					WithId("1").
					WithEmail("user@fake.com").
					WithPassword("$2a$10$DMu6hB30jb9SfUiNszbkzufXqeCgFFJPbMQeY5VpYNcYbWC.ZUB6a")

				m.EXPECT().Create(ctx, gomock.Any()).Return(got, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Failed",
			req: account.RegisterRequest{
				Email:    "user@fake.com",
				Password: "password",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().Create(ctx, gomock.Any()).Return(nil, assert.AnError)
			},
			wantErr: assert.Error,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mock.NewMockRepository(ctrl)
			s := account.NewService(cfg, repo)

			tt.mockFn(repo)

			res, err := s.Register(ctx, tt.req)
			if err != nil && tt.wantErr(t, err) {
				return
			}

			assert.NotNil(t, res)
			assert.NotEmpty(t, res.AccountId)
		})
	}
}

func TestService_ChangePassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.AuthConfig{
		Secret:   "secret",
		TokenExp: 3,
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     account.ChangePasswordRequest
		mockFn  func(m *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			req: account.ChangePasswordRequest{
				AccountId:   "1",
				OldPassword: "password",
				NewPassword: "newpassword",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().
					ChangePassword(ctx, "1", "password", "newpassword").
					Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Failed",
			req: account.ChangePasswordRequest{
				AccountId:   "1",
				OldPassword: "password",
				NewPassword: "newpassword",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().
					ChangePassword(ctx, "1", "password", "newpassword").
					Return(assert.AnError)
			},
			wantErr: assert.Error,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mock.NewMockRepository(ctrl)
			s := account.NewService(cfg, repo)

			tt.mockFn(repo)

			res, err := s.ChangePassword(ctx, tt.req)
			if err != nil && tt.wantErr(t, err) {
				return
			}

			assert.NotNil(t, res)
			assert.NotEmpty(t, res.AccountId)
		})
	}
}
