package authx_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/pkg/authx"
)

func TestRequirementsSatisfied(t *testing.T) {
	t.Parallel()

	reader := authx.Identity{Scopes: []string{"profile:read"}}
	writer := authx.Identity{Scopes: []string{"profile:read", "profile:write"}}
	admin := authx.Identity{Scopes: []string{"profile:write"}, Roles: []string{"admin"}}

	tests := []struct {
		name string
		req  authx.Requirements
		id   authx.Identity
		want bool
	}{
		{"scope granted", authx.Scope("profile:read"), reader, true},
		{"scope missing", authx.Scope("profile:write"), reader, false},
		{"role granted", authx.Role("admin"), admin, true},
		{"role missing", authx.Role("admin"), writer, false},
		{"every scope of one term is required", authx.Scope("profile:read", "profile:write"), reader, false},
		{"every scope of one term present", authx.Scope("profile:read", "profile:write"), writer, true},
		{"AllOf requires both", authx.AllOf(authx.Scope("profile:write"), authx.Role("admin")), admin, true},
		{"AllOf denies when the role is absent", authx.AllOf(authx.Scope("profile:write"), authx.Role("admin")), writer, false},
		{"AnyOf accepts either", authx.AnyOf(authx.Scope("profile:read"), authx.Role("admin")), reader, true},
		{"AnyOf accepts the other", authx.AnyOf(authx.Scope("profile:read"), authx.Role("admin")), admin, true},
		{"AnyOf denies when neither holds", authx.AnyOf(authx.Scope("nope"), authx.Role("nope")), reader, false},
		{"AllOf distributes over a nested AnyOf", authx.AllOf(
			authx.Scope("profile:write"),
			authx.AnyOf(authx.Role("admin"), authx.Role("owner")),
		), admin, true},
		{"Authenticated accepts any identity", authx.Authenticated(), authx.Identity{Subject: "user-1"}, true},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.req.Satisfied(tt.id))
		})
	}
}

// The zero value is what an undeclared requirement looks like. It must
// authorize nobody, so a requirement that was never populated cannot
// accidentally admit everyone.
func TestEmptyRequirementsAuthorizeNobody(t *testing.T) {
	t.Parallel()

	var none authx.Requirements

	assert.False(t, none.Satisfied(authx.Identity{
		Subject: "user-1",
		Scopes:  []string{"profile:read", "profile:write"},
		Roles:   []string{"admin"},
	}), "an undeclared requirement must not admit even a fully privileged caller")
}

// An unsatisfiable argument must make the whole conjunction unsatisfiable
// rather than being quietly dropped from it.
func TestAllOfPropagatesAnUnsatisfiableTerm(t *testing.T) {
	t.Parallel()

	var impossible authx.Requirements

	req := authx.AllOf(authx.Scope("profile:read"), impossible)

	assert.False(t, req.Satisfied(authx.Identity{Scopes: []string{"profile:read"}}))
}

func TestRequirementsOpenAPIProjection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  authx.Requirements
		want []map[string][]string
	}{
		{
			name: "a conjunction of scopes is one alternative",
			req:  authx.AllOf(authx.Scope("a"), authx.Scope("b")),
			want: []map[string][]string{{"bearerAuth": {"a", "b"}}},
		},
		{
			name: "a disjunction of scopes is two alternatives",
			req:  authx.AnyOf(authx.Scope("a"), authx.Scope("b")),
			want: []map[string][]string{{"bearerAuth": {"a"}}, {"bearerAuth": {"b"}}},
		},
		{
			name: "a role documents only as requiring a token",
			req:  authx.Role("admin"),
			want: []map[string][]string{{"bearerAuth": {}}},
		},
		{
			name: "Authenticated documents as requiring a token",
			req:  authx.Authenticated(),
			want: []map[string][]string{{"bearerAuth": {}}},
		},
		{
			name: "an undeclared requirement documents nothing",
			req:  nil,
			want: nil,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.req.OpenAPI("bearerAuth"))
		})
	}
}

// A nil scope list would serialize as JSON null, which is not what OpenAPI
// means by "this scheme, no scopes".
func TestOpenAPIScopesAreNeverNil(t *testing.T) {
	t.Parallel()

	got := authx.Role("admin").OpenAPI("bearerAuth")

	assert.NotNil(t, got[0]["bearerAuth"], "an empty scope list must serialize as [] rather than null")
}

// Requirements are values that get combined and reused, so a cross product must
// not alias a backing array between the alternatives it produces.
func TestAllOfDoesNotAliasItsInputs(t *testing.T) {
	t.Parallel()

	base := authx.Scope("profile:read")
	req := authx.AllOf(base, authx.AnyOf(authx.Scope("x"), authx.Scope("y")))

	req[0].Scopes[1] = "mutated"

	assert.Equal(t, []string{"profile:read", "y"}, req[1].Scopes,
		"writing to one alternative must not reach another")
	assert.Equal(t, []string{"profile:read"}, base[0].Scopes,
		"writing to a derived alternative must not reach the input")
}

// The rendered form goes into the denial log line, which is the only place an
// operator can see what a 403 actually wanted.
func TestRequirementsString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  authx.Requirements
		want string
	}{
		{"single scope", authx.Scope("profile:read"), "scope:profile:read"},
		{"conjunction", authx.AllOf(authx.Scope("profile:write"), authx.Role("admin")), "scope:profile:write+role:admin"},
		{"disjunction", authx.AnyOf(authx.Scope("a"), authx.Role("b")), "scope:a or role:b"},
		{"any token", authx.Authenticated(), "authenticated"},
		{"undeclared", nil, "none satisfiable"},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.req.String())
		})
	}
}
