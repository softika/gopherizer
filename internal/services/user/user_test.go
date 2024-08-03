package user

import (
	"context"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"reflect"
	"testing"
	"tldw/internal/model"

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
	}
	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
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
	type fields struct {
		repo Repository
	}
	type args struct {
		ctx context.Context
		id  ulid.ULID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				repo: tt.fields.repo,
			}
			got, err := s.DeleteById(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeleteById() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetByEmail(t *testing.T) {
	type fields struct {
		repo Repository
	}
	type args struct {
		ctx   context.Context
		email string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Response
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				repo: tt.fields.repo,
			}
			got, err := s.GetByEmail(tt.args.ctx, tt.args.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetByEmail() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_GetById(t *testing.T) {
	type fields struct {
		repo Repository
	}
	type args struct {
		ctx context.Context
		id  ulid.ULID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Response
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Service{
				repo: tt.fields.repo,
			}
			got, err := s.GetById(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetById() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetById() got = %v, want %v", got, tt.want)
			}
		})
	}
}
