package oidcx_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/authx"
	"github.com/softika/gopherizer/pkg/errorx"
	"github.com/softika/gopherizer/pkg/oidcx"
	"github.com/softika/gopherizer/pkg/testinfra"
)

// newProvider starts a fake identity provider and the verifier that trusts it.
// mutate adjusts the configuration before the verifier is built.
func newProvider(t *testing.T, mutate func(*config.OIDCConfig)) (*testinfra.OIDCProvider, oidcx.Verifier) {
	t.Helper()

	provider, err := testinfra.RunOIDC()
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Shutdown() })

	cfg := provider.Config
	if mutate != nil {
		mutate(&cfg)
	}

	verifier, err := oidcx.Init(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, verifier)

	return provider, verifier
}

// Disabled must yield an untyped nil, not an interface holding a nil pointer:
// the router tests "is authentication on?" by comparing against nil, and a
// typed nil would answer yes while verifying nothing.
func TestInitDisabledReturnsNoVerifier(t *testing.T) {
	t.Parallel()

	verifier, err := oidcx.Init(t.Context(), config.OIDCConfig{Enabled: false})

	require.NoError(t, err)
	assert.Nil(t, verifier, "a disabled configuration must produce no verifier at all")
}

func TestInitRejectsIncompleteConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.OIDCConfig
	}{
		{"no issuer", config.OIDCConfig{Enabled: true, Audience: "api"}},
		{"no audience", config.OIDCConfig{Enabled: true, Issuer: "https://issuer.example"}},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := oidcx.Init(t.Context(), tt.cfg)

			assert.Error(t, err, "an incomplete configuration must fail at startup, not at the first request")
		})
	}
}

// An unreachable provider must fail the boot rather than produce a verifier
// that rejects every request while the process reports itself healthy.
func TestInitFailsWhenTheProviderIsUnreachable(t *testing.T) {
	t.Parallel()

	_, err := oidcx.Init(t.Context(), config.OIDCConfig{
		Enabled:  true,
		Issuer:   "http://127.0.0.1:1",
		Audience: "api",
		// Short, so the retry loop cannot outlast the test.
		DiscoveryTimeout: 300 * time.Millisecond,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "127.0.0.1:1", "the error must name the issuer that could not be reached")
}

func TestVerifyAcceptsAValidToken(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, nil)

	token, err := provider.Token(map[string]any{
		"sub":   "user-1",
		"scope": "profile:read profile:write",
		"roles": []string{"admin"},
	})
	require.NoError(t, err)

	id, err := verifier.Verify(t.Context(), token)

	require.NoError(t, err)
	assert.Equal(t, "user-1", id.Subject)
	assert.Equal(t, []string{"profile:read", "profile:write"}, id.Scopes)
	assert.Equal(t, []string{"admin"}, id.Roles)
	assert.Equal(t, "user-1", id.Claims["sub"], "the raw claims stay available for a service that needs one")
}

// RS256 must work alongside the ES256 default, or the pinned algorithm list is
// quietly RS256-only in one direction and ES256-only in the other.
func TestVerifyAcceptsRS256(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, nil)

	token, err := provider.RSAToken(map[string]any{"sub": "user-1"})
	require.NoError(t, err)

	id, err := verifier.Verify(t.Context(), token)

	require.NoError(t, err)
	assert.Equal(t, "user-1", id.Subject)
}

func TestVerifyRejectsBadTokens(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, nil)

	foreignKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name   string
		reason string
		mint   func() (string, error)
	}{
		{
			name:   "wrong audience",
			reason: "a token minted for another api must not be accepted by this one",
			mint:   func() (string, error) { return provider.Token(map[string]any{"aud": "another-api"}) },
		},
		{
			name:   "wrong issuer",
			reason: "a token from an untrusted issuer must be rejected even when well formed",
			mint:   func() (string, error) { return provider.Token(map[string]any{"iss": "https://evil.example"}) },
		},
		{
			name:   "expired",
			reason: "an expired token must not be honoured",
			mint: func() (string, error) {
				return provider.Token(map[string]any{"exp": time.Now().Add(-time.Hour).Unix()})
			},
		},
		{
			name:   "unsigned",
			reason: "alg:none is the oldest jwt bypass and must never verify",
			mint:   func() (string, error) { return provider.UnsignedToken(nil) },
		},
		{
			name:   "signed by an unknown key",
			reason: "a valid signature from a key the provider does not publish proves nothing",
			mint: func() (string, error) {
				return provider.TokenSignedWith(jose.ES256, foreignKey, "unknown-key", nil)
			},
		},
		{
			name:   "hmac signed",
			reason: "a symmetric algorithm must be unrepresentable, so key confusion cannot arise",
			mint: func() (string, error) {
				return provider.TokenSignedWith(jose.HS256, []byte("0123456789abcdef0123456789abcdef"), "test-key", nil)
			},
		},
		{
			name:   "carries at_hash",
			reason: "at_hash marks an id token, which must never pass as an access token",
			mint:   func() (string, error) { return provider.Token(map[string]any{"at_hash": "abc"}) },
		},
		{
			name:   "carries c_hash",
			reason: "c_hash marks an id token, which must never pass as an access token",
			mint:   func() (string, error) { return provider.Token(map[string]any{"c_hash": "abc"}) },
		},
		{
			name:   "carries nonce",
			reason: "nonce marks an id token from an interactive flow",
			mint:   func() (string, error) { return provider.Token(map[string]any{"nonce": "abc"}) },
		},
		{
			name:   "no subject",
			reason: "an identity nothing can be attributed to is not an identity",
			mint:   func() (string, error) { return provider.Token(map[string]any{"sub": ""}) },
		},
		{
			name:   "not a jwt",
			reason: "arbitrary header content must not reach the verifier as a credential",
			mint:   func() (string, error) { return "not-a-token", nil },
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, mintErr := tt.mint()
			require.NoError(t, mintErr)

			_, err := verifier.Verify(t.Context(), token)

			require.Error(t, err, tt.reason)
			assert.Equal(t, errorx.ErrUnauthorized, errorx.TypeOf(err),
				"a refused credential must read as unauthorized, not as a server fault")
		})
	}
}

func TestVerifyRequiresTypedAccessTokenWhenConfigured(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, func(cfg *config.OIDCConfig) {
		cfg.RequireTypedAccessToken = true
	})

	tests := []struct {
		name    string
		typ     string
		wantErr bool
	}{
		{"rfc 9068 type", "at+jwt", false},
		{"media type form", "application/at+jwt", false},
		{"uppercase is still the same media type", "AT+JWT", false},
		{"keycloak default", "Bearer", true},
		{"id token", "JWT", true},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token, err := provider.TokenWithType(tt.typ, map[string]any{"sub": "user-1"})
			require.NoError(t, err)

			_, err = verifier.Verify(t.Context(), token)

			if tt.wantErr {
				assert.Error(t, err, "typ %q must not be accepted as an access token", tt.typ)
				return
			}
			assert.NoError(t, err, "typ %q is a valid rfc 9068 access token", tt.typ)
		})
	}
}

// Off by default, because Keycloak stamps `typ: Bearer` unless the client opts
// in and a template that rejected those out of the box would look broken.
func TestVerifyIgnoresTokenTypeByDefault(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, nil)

	token, err := provider.TokenWithType("Bearer", map[string]any{"sub": "user-1"})
	require.NoError(t, err)

	_, err = verifier.Verify(t.Context(), token)

	assert.NoError(t, err)
}

func TestVerifyReadsProviderSpecificClaimShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		scopeClaim string
		rolesClaim string
		claims     map[string]any
		wantScopes []string
		wantRoles  []string
	}{
		{
			name:       "space delimited scope string",
			scopeClaim: "scope",
			rolesClaim: "roles",
			claims:     map[string]any{"scope": "a b", "roles": []string{"admin"}},
			wantScopes: []string{"a", "b"},
			wantRoles:  []string{"admin"},
		},
		{
			name:       "entra id scp array",
			scopeClaim: "scp",
			rolesClaim: "roles",
			claims:     map[string]any{"scp": []string{"a", "b"}},
			wantScopes: []string{"a", "b"},
		},
		{
			name:       "keycloak nested realm roles",
			scopeClaim: "scope",
			rolesClaim: "realm_access.roles",
			claims:     map[string]any{"realm_access": map[string]any{"roles": []string{"admin", "user"}}},
			wantRoles:  []string{"admin", "user"},
		},
		{
			name:       "auth0 namespaced claim containing dots",
			scopeClaim: "scope",
			rolesClaim: "https://example.com/roles",
			claims:     map[string]any{"https://example.com/roles": []string{"editor"}},
			wantRoles:  []string{"editor"},
		},
		{
			name:       "absent claims yield nothing rather than failing",
			scopeClaim: "scope",
			rolesClaim: "realm_access.roles",
			claims:     map[string]any{},
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, verifier := newProvider(t, func(cfg *config.OIDCConfig) {
				cfg.ScopeClaim = tt.scopeClaim
				cfg.RolesClaim = tt.rolesClaim
			})

			claims := map[string]any{"sub": "user-1"}
			for k, v := range tt.claims {
				claims[k] = v
			}

			token, err := provider.Token(claims)
			require.NoError(t, err)

			id, err := verifier.Verify(t.Context(), token)

			require.NoError(t, err)
			assert.Equal(t, tt.wantScopes, id.Scopes)
			assert.Equal(t, tt.wantRoles, id.Roles)
		})
	}
}

// A verified identity must be usable through authx without the api layer
// needing to know how it was produced.
func TestVerifiedIdentityAnswersRequirements(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, nil)

	token, err := provider.Token(map[string]any{
		"sub":   "user-1",
		"scope": "profile:write",
		"roles": []string{"admin"},
	})
	require.NoError(t, err)

	id, err := verifier.Verify(t.Context(), token)
	require.NoError(t, err)

	assert.True(t, authx.AllOf(authx.Scope("profile:write"), authx.Role("admin")).Satisfied(id))
	assert.False(t, authx.Scope("profile:delete").Satisfied(id))
}

// An identity provider that cannot be reached is its outage, not a fleet of
// callers all presenting bad tokens at the same instant. Answering 401 for it
// sends the investigation to the clients.
//
// These run against a genuinely broken provider rather than a stub. A stub was
// what let the original defect ship: go-oidc flattens the transport error into
// a string with %v, so the chain a hand-built error preserves is exactly the
// one the real library destroys.
func TestVerifyReportsAProviderOutageAsUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		brk  func(*testinfra.OIDCProvider)
	}{
		{
			name: "unreachable",
			brk:  func(p *testinfra.OIDCProvider) { _ = p.Shutdown() },
		},
		{
			name: "key endpoint failing",
			brk:  func(p *testinfra.OIDCProvider) { p.FailJWKS(http.StatusServiceUnavailable) },
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, verifier := newProvider(t, nil)

			token, err := provider.Token(map[string]any{"sub": "user-1"})
			require.NoError(t, err)

			// Broken before the first verification, so no key set is cached and
			// the fetch genuinely has to happen.
			tt.brk(provider)

			_, err = verifier.Verify(t.Context(), token)

			require.Error(t, err)
			assert.Equal(t, errorx.ErrUnavailable, errorx.TypeOf(err),
				"a provider outage must answer 503, not 401: %v", err)
		})
	}
}

// A running process keeps serving through an outage, because the key set is
// cached and refetched only for a key id it has not seen.
func TestVerifySurvivesAnOutageOnceKeysAreCached(t *testing.T) {
	t.Parallel()

	provider, verifier := newProvider(t, nil)

	token, err := provider.Token(map[string]any{"sub": "user-1"})
	require.NoError(t, err)

	_, err = verifier.Verify(t.Context(), token)
	require.NoError(t, err, "the first call warms the key set")

	provider.FailJWKS(http.StatusServiceUnavailable)

	later, err := provider.Token(map[string]any{"sub": "user-2"})
	require.NoError(t, err)

	id, err := verifier.Verify(t.Context(), later)

	require.NoError(t, err, "a cached key set must outlive the provider")
	assert.Equal(t, "user-2", id.Subject)
}
