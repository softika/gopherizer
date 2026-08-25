package profile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/softika/gopherizer/internal/profile"
	"github.com/softika/gopherizer/internal/profile/mock"
	"github.com/softika/gopherizer/pkg/errorx"
)

// The service must pass the repository's error type through rather than
// inventing one. Hard-coding a type here is what turned a database outage into
// a 404 and a missing row into a 500.
func TestService_PreservesRepositoryErrorType(t *testing.T) {
	t.Parallel()

	notFound := errorx.NewError(errors.New("no rows in result set"), errorx.ErrNotFound)
	outage := errors.New("failed to connect to host=db port=5432")

	tests := []struct {
		name     string
		repoErr  error
		wantType errorx.ErrorType
	}{
		{
			name:     "missing row stays not-found",
			repoErr:  notFound,
			wantType: errorx.ErrNotFound,
		},
		{
			name:     "database outage is internal, not not-found",
			repoErr:  outage,
			wantType: errorx.ErrInternal,
		},
	}

	for _, tc := range tests {
		tt := tc

		t.Run("GetById/"+tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockRepository(ctrl)
			repo.EXPECT().GetById(gomock.Any(), gomock.Any()).Return(nil, tt.repoErr)

			_, err := profile.NewService(repo).GetById(context.Background(), profile.GetRequest{Id: "id"})

			assert.Error(t, err)
			assert.Equal(t, tt.wantType, errorx.TypeOf(err))
		})

		t.Run("Update/"+tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockRepository(ctrl)
			repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil, tt.repoErr)

			_, err := profile.NewService(repo).Update(context.Background(), profile.UpdateRequest{Id: "id"})

			assert.Error(t, err)
			assert.Equal(t, tt.wantType, errorx.TypeOf(err))
		})

		t.Run("DeleteById/"+tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mock.NewMockRepository(ctrl)
			repo.EXPECT().DeleteById(gomock.Any(), gomock.Any()).Return(tt.repoErr)

			ok, err := profile.NewService(repo).DeleteById(context.Background(), profile.DeleteRequest{Id: "id"})

			assert.Error(t, err)
			assert.False(t, ok, "a failed delete must not report success")
			assert.Equal(t, tt.wantType, errorx.TypeOf(err))
		})
	}
}
