package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/database/repositories"
	"github.com/softika/gopherizer/internal/profile"
	"github.com/softika/gopherizer/pkg/errorx"
)

// missingId is a well-formed uuid that is guaranteed not to exist.
const missingId = "00000000-0000-0000-0000-000000000000"

// The repository owns the driver, so it is where a "no rows" result must be
// translated into a domain error type. Without this the API cannot tell a
// missing row apart from a database outage.
func (s *RepositoriesTestSuite) TestProfileRepository_NotFoundIsTyped() {
	repo := repositories.NewProfileRepository(s.dbService)
	ctx := s.dbContainer.Ctx

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "GetById on a missing row",
			call: func() error {
				_, err := repo.GetById(ctx, missingId)
				return err
			},
		},
		{
			name: "Update on a missing row",
			call: func() error {
				_, err := repo.Update(ctx, profile.New().
					WithId(missingId).
					WithFirstName("Ghost").
					WithLastName("Profile"))
				return err
			},
		},
		{
			name: "DeleteById on a missing row",
			call: func() error {
				return repo.DeleteById(ctx, missingId)
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			err := tt.call()

			assert.Error(t, err, "a missing row must be reported as an error")
			assert.Equal(t, errorx.ErrNotFound, errorx.TypeOf(err),
				"expected ErrNotFound, got type %v for: %v", errorx.TypeOf(err), err)
		})
	}
}

// A real failure must not be mislabelled as "not found".
func (s *RepositoriesTestSuite) TestProfileRepository_RealErrorIsNotNotFound() {
	repo := repositories.NewProfileRepository(s.dbService)

	// An invalid uuid is a driver-level failure, not a missing row.
	_, err := repo.GetById(s.dbContainer.Ctx, "not-a-uuid")

	assert.Error(s.T(), err)
	assert.NotEqual(s.T(), errorx.ErrNotFound, errorx.TypeOf(err),
		"a malformed input must not be reported as not-found")
}
