package user

import (
	"context"
	"testing"
	"tldw/internal/model"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"tldw/internal/services/user/mock"
)

func TestService_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	req := func() CreateRequest {
		return CreateRequest{
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john.doe@fake.com",
			Password:  "password",
		}
	}

	tests := []struct {
		name    string
		req     CreateRequest
		mockFn  func(r *mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  req(),
			mockFn: func(r *mock.MockRepository) {
				u := model.NewUser().
					WithFirstName("John").
					WithLastName("Doe").
					WithEmail("john.doe@fake.com")

				r.EXPECT().Create(ctx, gomock.Any()).Return(u, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  req(),
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().Create(ctx, gomock.Any()).Return(nil, assert.AnError)
			},
			wantErr: assert.Error,
		},
	}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			repo := mock.NewMockRepository(ctrl)
			s := NewService(repo)
			tt.mockFn(repo)

			// when
			got, err := s.Create(ctx, tt.req)

			// then
			if err != nil && tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.req.Email, got.Email)
			assert.Equal(t, tt.req.FirstName, got.FirstName)
			assert.Equal(t, tt.req.LastName, got.LastName)
		})
	}
}

func TestService_DeleteById(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		mockFn  func(r *mock.MockRepository)
		req     ulid.ULID
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  ulid.Make(),
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().DeleteById(ctx, gomock.Any()).Return(nil)
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  ulid.Make(),
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().DeleteById(ctx, gomock.Any()).Return(assert.AnError)
			},
			want:    false,
			wantErr: assert.Error,
		},
	}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			repo := mock.NewMockRepository(ctrl)
			s := NewService(repo)
			tt.mockFn(repo)

			// when
			got, err := s.DeleteById(ctx, tt.req)

			// then
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_GetByEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     string
		mockFn  func(r *mock.MockRepository)
		want    *Response
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  "fake@email.com",
			mockFn: func(r *mock.MockRepository) {
				u := model.NewUser().
					WithFirstName("John").
					WithLastName("Doe").
					WithEmail("fake@email.com")
				r.EXPECT().GetByEmail(ctx, gomock.Any()).Return(u, nil)
			},
			want: &Response{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "fake@email.com",
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  "fake@email.com",
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().GetByEmail(ctx, gomock.Any()).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			repo := mock.NewMockRepository(ctrl)
			s := NewService(repo)
			tt.mockFn(repo)

			// when
			got, err := s.GetByEmail(ctx, tt.req)

			// then
			if err != nil && tt.wantErr(t, err) {
				return
			}

			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.FirstName, got.FirstName)
			assert.Equal(t, tt.want.LastName, got.LastName)
		})
	}
}

func TestService_GetById(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	tests := []struct {
		name    string
		req     ulid.ULID
		mockFn  func(r *mock.MockRepository)
		want    *Response
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  ulid.Make(),
			mockFn: func(r *mock.MockRepository) {
				u := model.NewUser().
					WithFirstName("John").
					WithLastName("Doe").
					WithEmail("fake@email.com")
				r.EXPECT().GetById(ctx, gomock.Any()).Return(u, nil)
			},
			want: &Response{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "fake@email.com",
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  ulid.Make(),
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().GetById(ctx, gomock.Any()).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// given
			repo := mock.NewMockRepository(ctrl)
			s := NewService(repo)
			tt.mockFn(repo)

			// when
			got, err := s.GetById(ctx, tt.req)

			// then
			if err != nil && tt.wantErr(t, err) {
				return
			}

			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.FirstName, got.FirstName)
			assert.Equal(t, tt.want.LastName, got.LastName)
		})
	}
}
