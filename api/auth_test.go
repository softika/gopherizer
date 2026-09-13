package api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/pkg/authx"
	"github.com/softika/gopherizer/pkg/errorx"
)

const testToken = "a-valid-looking-token-value"

// stubVerifier stands in for oidcx so these tests exercise the guard rather
// than token verification, which pkg/oidcx already covers against a real
// provider.
type stubVerifier struct {
	id  authx.Identity
	err error
}

func (s stubVerifier) Verify(_ context.Context, raw string) (authx.Identity, error) {
	if s.err != nil {
		return authx.Identity{}, s.err
	}
	if raw != testToken {
		return authx.Identity{}, errorx.NewError(errors.New("unknown token"), errorx.ErrUnauthorized)
	}

	return s.id, nil
}

// guardedAPI registers one secured operation and one public one, so every test
// can check that a rule applies where it was declared and nowhere else.
func guardedAPI(t *testing.T, v stubVerifier, req authx.Requirements) (humatest.TestAPI, *bytes.Buffer) {
	t.Helper()

	_, api := humatest.New(t)

	logger, buf := testLogger()
	g := newGuard(api, v, logger)

	huma.Register(api, secured(huma.Operation{
		OperationID: "secured",
		Method:      http.MethodPost,
		Path:        "/secured",
	}, req, g), func(ctx context.Context, in *struct {
		Body struct {
			Name string `json:"name"`
		}
	}) (*struct{ Body map[string]string }, error) {
		id, ok := authx.FromContext(ctx)
		require.True(t, ok, "a secured handler must see the identity the guard resolved")

		return &struct{ Body map[string]string }{Body: map[string]string{"subject": id.Subject}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "public",
		Method:      http.MethodGet,
		Path:        "/public",
	}, func(context.Context, *struct{}) (*struct{ Body map[string]string }, error) {
		return &struct{ Body map[string]string }{Body: map[string]string{"ok": "true"}}, nil
	})

	return api, buf
}

func TestBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{"canonical", "Bearer abc", "abc", true},
		{"scheme is case insensitive per rfc 7235", "bearer abc", "abc", true},
		{"surrounding space is trimmed", "Bearer   abc  ", "abc", true},
		{"absent", "", "", false},
		{"another scheme", "Basic dXNlcjpwYXNz", "", false},
		{"scheme with no credential", "Bearer ", "", false},
		{"scheme only", "Bearer", "", false},
		{"bare token with no scheme", "abc", "", false},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, ok := bearerToken(tt.header)

			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantToken, token)
		})
	}
}

func TestGuardAcceptsASatisfiedRequirement(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{id: authx.Identity{Subject: "user-1", Scopes: []string{"profile:write"}}},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})

	require.Equal(t, http.StatusOK, res.Code)
	assert.Contains(t, res.Body.String(), "user-1", "the handler must be able to read the resolved subject")
}

func TestGuardRejectsMissingAndMalformedCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
	}{
		{"no header at all", ""},
		{"a different scheme", "Authorization: Basic dXNlcjpwYXNz"},
		{"bearer with no token", "Authorization: Bearer "},
		{"an unknown token", "Authorization: Bearer not-the-right-token"},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			api, _ := guardedAPI(t,
				stubVerifier{id: authx.Identity{Subject: "user-1", Scopes: []string{"profile:write"}}},
				authx.Scope("profile:write"),
			)

			args := []any{map[string]string{"name": "grace"}}
			if tt.header != "" {
				args = append([]any{tt.header}, args...)
			}

			res := api.Post("/secured", args...)

			assert.Equal(t, http.StatusUnauthorized, res.Code)
			assert.Equal(t, `Bearer error="invalid_token"`, res.Header().Get("WWW-Authenticate"),
				"rfc 7235 requires a challenge alongside a 401")
		})
	}
}

// A verified caller who lacks the requirement is a different answer from one
// who presented nothing: 403 says the credential was accepted and still did not
// suffice, which is what tells a client that retrying is pointless.
func TestGuardRejectsAnUnsatisfiedRequirement(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{id: authx.Identity{Subject: "user-1", Scopes: []string{"profile:read"}}},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})

	assert.Equal(t, http.StatusForbidden, res.Code)
	assert.Empty(t, res.Header().Get("WWW-Authenticate"),
		"a challenge invites another credential, which is the wrong advice when this one was accepted")
}

// An identity provider that cannot be reached is the provider's outage, not a
// fleet of clients suddenly holding bad tokens. Reporting it as 401 sends an
// investigation to the wrong place.
func TestGuardReportsAnUnreachableProviderAsUnavailable(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{err: errorx.NewError(errors.New("jwks fetch failed"), errorx.ErrUnavailable)},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})

	assert.Equal(t, http.StatusServiceUnavailable, res.Code)
}

// An error carrying no domain type is a bug in the verifier, not a verdict on
// the credential. Reading it charitably as a rejection would hide the bug; it
// must fail closed as a server fault instead.
func TestGuardFailsClosedOnAnUntypedVerifierError(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{err: errors.New("boom")},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})

	assert.Equal(t, http.StatusInternalServerError, res.Code)
}

// An operation that declares no requirement must stay reachable, which is what
// keeps the health probes open without a path allowlist.
func TestGuardLeavesUndeclaredOperationsOpen(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{id: authx.Identity{Subject: "user-1"}},
		authx.Scope("profile:write"),
	)

	res := api.Get("/public")

	assert.Equal(t, http.StatusOK, res.Code)
}

// A nil guard is how "authentication is off" is spelled, and it must leave the
// operation both open and undocumented rather than half-applied.
func TestSecuredIsInertWithoutAGuard(t *testing.T) {
	t.Parallel()

	op := secured(huma.Operation{OperationID: "x"}, authx.Scope("profile:write"), nil)

	assert.Nil(t, op.Security, "an unenforced operation must not advertise a scheme")
	assert.Empty(t, op.Middlewares)
}

// A requirement that nobody can satisfy must refuse everybody -- never become
// a public endpoint.
//
// The two halves are individually reasonable and compose into an inversion:
// AllOf reports an unsatisfiable conjunction as an empty Requirements, and an
// empty Requirements once meant "no requirement declared". Satisfied said
// nobody; secured said everybody. A fork writing
// secured(op, authx.AnyOf(rolesFromConfig...), g) against an empty config list
// would have published the endpoint.
func TestSecuredEnforcesAnUnsatisfiableRequirement(t *testing.T) {
	t.Parallel()

	_, api := humatest.New(t)
	logger, _ := testLogger()
	g := newGuard(api, stubVerifier{}, logger)

	tests := []struct {
		name string
		req  authx.Requirements
	}{
		{"a conjunction with an unsatisfiable term", authx.AllOf(authx.Scope("profile:write"), nil)},
		{"an empty disjunction", authx.AnyOf()},
		{"an explicit nil", nil},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op := secured(huma.Operation{OperationID: "x"}, tt.req, g)

			assert.NotEmpty(t, op.Middlewares, "an unsatisfiable requirement must still be enforced")
			assert.NotEmpty(t, op.Security, "and must not be documented as open")
		})
	}
}

// The same, end to end: a fully privileged caller is still refused.
func TestGuardRefusesEveryoneWhenNothingIsSatisfiable(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{id: authx.Identity{
			Subject: "superuser",
			Scopes:  []string{"profile:read", "profile:write"},
			Roles:   []string{"admin"},
		}},
		authx.AllOf(authx.Scope("profile:write"), nil),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})

	assert.Equal(t, http.StatusForbidden, res.Code,
		"a rule nobody can satisfy must refuse even a fully privileged caller, not admit anonymous ones")
}

// Rejection has to happen before the body is validated, or an anonymous caller
// can still make the service parse and check whatever it sent.
//
// Proven by what comes back rather than by watching the reader: huma answers
// 422 for a body that fails validation, so a malformed body that still draws a
// 401 shows the guard ran first and the validator never did.
func TestGuardRejectsBeforeTheBodyIsValidated(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{id: authx.Identity{Subject: "user-1"}},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Content-Type: application/json", strings.NewReader(`{"name": 12345}`))

	assert.Equal(t, http.StatusUnauthorized, res.Code,
		"an unauthenticated request must be refused before its body is validated, not answered 422")
}

func TestGuardLogsTheDecisionWithoutTheCredential(t *testing.T) {
	t.Parallel()

	api, buf := guardedAPI(t,
		stubVerifier{id: authx.Identity{Subject: "user-1", Scopes: []string{"profile:read"}}},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})
	require.Equal(t, http.StatusForbidden, res.Code)

	logged := buf.String()

	assert.Contains(t, logged, "request denied")
	assert.Contains(t, logged, "user-1", "an operator needs to know which principal was refused")
	assert.NotContains(t, logged, testToken, "the credential must never reach the log")
}

// The body must carry the same generic wording as every other error of its
// type: naming the failed check tells an attacker which part they have right.
func TestGuardDoesNotExplainWhyItRefused(t *testing.T) {
	t.Parallel()

	api, _ := guardedAPI(t,
		stubVerifier{id: authx.Identity{Subject: "user-1", Scopes: []string{"profile:read"}}},
		authx.Scope("profile:write"),
	)

	res := api.Post("/secured", "Authorization: Bearer "+testToken, map[string]string{"name": "grace"})

	body := res.Body.String()
	assert.Contains(t, body, clientMessages[errorx.ErrForbidden])
	assert.NotContains(t, body, "profile:write", "the required scope must not be disclosed")
	assert.NotContains(t, body, "requirement not satisfied", "the internal reason must stay in the log")
}
