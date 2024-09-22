package account

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"tldw/config"
	"tldw/internal/model"
	"tldw/internal/services/account/mock"
)

func TestService_Login(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cfg := config.Auth{
		Secret:   "secret",
		TokenExp: 3,
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     LoginRequest
		mockFn  func(m *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			req: LoginRequest{
				Email:    "user@fake.com",
				Password: "password",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().GetByEmail(ctx, "user@fake.com").Return(&model.Identity{
					Email:    "user@fake.com",
					Password: "$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq",
				}, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid credentials",
			req: LoginRequest{
				Email:    "user@fake.com",
				Password: "invalid",
			},
			mockFn: func(m *mock.MockRepository) {
				m.EXPECT().GetByEmail(ctx, "user@fake.com").Return(&model.Identity{
					Email:    "user@fake.com",
					Password: "$2a$10$.2/hbR6YIEfp4a7zvZ7xpO0fUUySsjM6.wgH0aWuqFN/sJPR5uEFq",
				}, nil)
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid Email",
			req: LoginRequest{
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
			s := NewService(cfg, repo)

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

	cfg := config.Auth{
		Secret:   "secret",
		TokenExp: 3,
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     RegisterRequest
		mockFn  func(m *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			req: RegisterRequest{
				Email:    "user@fake.com",
				Password: "password",
			},
			mockFn: func(m *mock.MockRepository) {
				got := model.NewAccount().
					WithId("1").
					WithEmail("user@fake.com").
					WithPassword("$2a$10$DMu6hB30jb9SfUiNszbkzufXqeCgFFJPbMQeY5VpYNcYbWC.ZUB6a")

				m.EXPECT().Create(ctx, gomock.Any()).Return(got, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "Failed",
			req: RegisterRequest{
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
			s := NewService(cfg, repo)

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

	cfg := config.Auth{
		Secret:   "secret",
		TokenExp: 3,
	}

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     ChangePasswordRequest
		mockFn  func(m *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Success",
			req: ChangePasswordRequest{
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
			req: ChangePasswordRequest{
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
			s := NewService(cfg, repo)

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
