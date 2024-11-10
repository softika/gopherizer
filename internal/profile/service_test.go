package profile_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/softika/gopherizer/internal/profile"
	"github.com/softika/gopherizer/internal/profile/mock"
)

func TestService_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	req := func() profile.CreateRequest {
		return profile.CreateRequest{
			FirstName: "John",
			LastName:  "Doe",
		}
	}

	tests := []struct {
		name    string
		req     profile.CreateRequest
		mockFn  func(*mock.MockRepository)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  req(),
			mockFn: func(r *mock.MockRepository) {
				u := profile.New().
					WithFirstName("John").
					WithLastName("Doe")

				r.EXPECT().
					Create(ctx, gomock.Any()).
					Return(u, nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  req(),
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().
					Create(ctx, gomock.Any()).
					Return(nil, assert.AnError)
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
			s := profile.NewService(repo)

			tt.mockFn(repo)

			// when
			got, err := s.Create(ctx, tt.req)

			// then
			if err != nil && tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.req.FirstName, got.FirstName)
			assert.Equal(t, tt.req.LastName, got.LastName)
		})
	}
}

func TestService_DeleteById(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	id := "b8c22ea5-0d76-4abc-8ff2-5a31bb4daddc"

	tests := []struct {
		name    string
		mockFn  func(r *mock.MockRepository)
		req     string
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  id,
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().DeleteById(ctx, id).Return(nil)
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  id,
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().DeleteById(ctx, id).Return(assert.AnError)
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
			s := profile.NewService(repo)
			tt.mockFn(repo)

			// when
			got, err := s.DeleteById(ctx, tt.req)

			// then
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_GetById(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	ctrl := gomock.NewController(t)

	id := "b8c22ea5-0d76-4abc-8ff2-5a31bb4daddc"

	tests := []struct {
		name    string
		req     string
		mockFn  func(r *mock.MockRepository)
		want    *profile.Response
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			req:  id,
			mockFn: func(r *mock.MockRepository) {
				u := profile.New().
					WithFirstName("John").
					WithLastName("Doe")
				r.EXPECT().GetById(ctx, id).Return(u, nil)
			},
			want: &profile.Response{
				FirstName: "John",
				LastName:  "Doe",
			},
			wantErr: assert.NoError,
		},
		{
			name: "error",
			req:  id,
			mockFn: func(r *mock.MockRepository) {
				r.EXPECT().GetById(ctx, id).Return(nil, assert.AnError)
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
			s := profile.NewService(repo)
			tt.mockFn(repo)

			// when
			got, err := s.GetById(ctx, tt.req)

			// then
			if err != nil && tt.wantErr(t, err) {
				return
			}

			assert.Equal(t, tt.want.FirstName, got.FirstName)
			assert.Equal(t, tt.want.LastName, got.LastName)
		})
	}
}
