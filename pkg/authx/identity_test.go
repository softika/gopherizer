package authx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/pkg/authx"
)

func TestIdentityCarriesScopesAndRoles(t *testing.T) {
	t.Parallel()

	id := authx.Identity{
		Subject: "user-1",
		Scopes:  []string{"profile:read"},
		Roles:   []string{"admin"},
	}

	assert.True(t, id.HasScope("profile:read"))
	assert.False(t, id.HasScope("profile:write"), "a scope that was not granted must not be reported")
	assert.True(t, id.HasRole("admin"))
	assert.False(t, id.HasRole("auditor"), "a role that was not granted must not be reported")
}

// The zero Identity is what a bug produces, so it must grant nothing rather
// than panic on a nil slice.
func TestZeroIdentityGrantsNothing(t *testing.T) {
	t.Parallel()

	var id authx.Identity

	assert.False(t, id.HasScope("profile:read"))
	assert.False(t, id.HasRole("admin"))
}

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()

	want := authx.Identity{Subject: "user-1", Scopes: []string{"profile:read"}}

	got, ok := authx.FromContext(authx.NewContext(t.Context(), want))

	assert.True(t, ok)
	assert.Equal(t, want, got)
}

// An unauthenticated request must be distinguishable from one that
// authenticated as a principal holding nothing. A handler that conflates the
// two grants anonymous callers whatever it meant to grant the empty ones.
func TestFromContextReportsAbsence(t *testing.T) {
	t.Parallel()

	got, ok := authx.FromContext(context.Background())

	assert.False(t, ok, "a bare context carries no identity")
	assert.Equal(t, authx.Identity{}, got)

	empty, ok := authx.FromContext(authx.NewContext(t.Context(), authx.Identity{Subject: "user-1"}))

	assert.True(t, ok, "a scopeless identity is still an identity")
	assert.Equal(t, "user-1", empty.Subject)
}
