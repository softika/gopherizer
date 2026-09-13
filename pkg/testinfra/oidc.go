package testinfra

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/softika/gopherizer/config"
)

const (
	oidcAudience = "gopherizer-test"
	oidcKeyID    = "test-key"
	jwksPath     = "/jwks.json"
)

// OIDCProvider is an in-process stand-in for an identity provider.
//
// It exists because the suite runs under a 30 second timeout and must pass with
// no Docker daemon and no network: a real provider in a container costs more
// than the whole rest of the suite, and a shared remote one makes the tests
// dependent on somebody else's uptime. Everything a resource server actually
// reads -- the discovery document, the key set, and signed tokens -- is served
// from this process.
//
// It signs with ECDSA P-256 by default because key generation is measured in
// microseconds, where RSA-2048 is measured in tens of milliseconds. That also
// makes the default path exercise ES256, proving the verifier's pinned
// algorithm list is not quietly RS256-only.
type OIDCProvider struct {
	Issuer   string
	Audience string
	// Config is ready to hand to oidcx.Init, mirroring how PostgresContainer
	// exposes a DatabaseConfig.
	Config   config.OIDCConfig
	Shutdown func() error

	key *ecdsa.PrivateKey
	// The RSA key is generated only if a test asks for RS256, and the key
	// set advertises it only once one has. Generating it for every provider
	// would put tens of milliseconds on every test that never needs it.
	rsaOnce   func() *rsa.PrivateKey
	rsaWanted atomic.Bool
	// jwksStatus, when non-zero, makes the key endpoint answer with it instead
	// of the key set -- a provider that is up but failing, which a resource
	// server has to tell apart from a caller presenting a bad token.
	jwksStatus atomic.Int64
}

// FailJWKS makes the key endpoint answer with status instead of the key set.
// Passing 0 restores it.
func (p *OIDCProvider) FailJWKS(status int) {
	p.jwksStatus.Store(int64(status))
}

// RunOIDC starts a provider serving discovery and a key set.
func RunOIDC() (*OIDCProvider, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a signing key: %w", err)
	}

	p := &OIDCProvider{
		Audience: oidcAudience,
		key:      key,
		rsaOnce: sync.OnceValue(func() *rsa.PrivateKey {
			rsaKey, rsaErr := rsa.GenerateKey(rand.Reader, 2048)
			if rsaErr != nil {
				panic("failed to generate an rsa signing key: " + rsaErr.Error())
			}

			return rsaKey
		}),
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// The handlers close over the server so the discovery document can report
	// the address it is actually served from. go-oidc compares the configured
	// issuer against this value byte for byte, so it cannot be built before
	// the listener has an address.
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                                server.URL,
			"jwks_uri":                              server.URL + jwksPath,
			"authorization_endpoint":                server.URL + "/authorize",
			"token_endpoint":                        server.URL + "/token",
			"id_token_signing_alg_values_supported": []string{"ES256", "RS256"},
		})
	})

	mux.HandleFunc(jwksPath, func(w http.ResponseWriter, _ *http.Request) {
		if status := p.jwksStatus.Load(); status != 0 {
			http.Error(w, "key set unavailable", int(status))

			return
		}

		keys := []jose.JSONWebKey{{
			Key: key.Public(), KeyID: oidcKeyID, Algorithm: string(jose.ES256), Use: "sig",
		}}
		// go-oidc refetches the key set when it meets a key id it does not
		// know, so a token minted after the first fetch still verifies.
		if p.rsaWanted.Load() {
			keys = append(keys, jose.JSONWebKey{
				Key: p.rsaOnce().Public(), KeyID: oidcKeyID + "-rsa", Algorithm: string(jose.RS256), Use: "sig",
			})
		}

		writeJSON(w, jose.JSONWebKeySet{Keys: keys})
	})

	p.Issuer = server.URL
	p.Config = config.OIDCConfig{
		Enabled:          true,
		Issuer:           server.URL,
		Audience:         oidcAudience,
		ScopeClaim:       "scope",
		RolesClaim:       "roles",
		DiscoveryTimeout: 5 * time.Second,
	}
	p.Shutdown = func() error {
		server.Close()

		return nil
	}

	return p, nil
}

// Token signs claims with the provider's key, filling in any of iss, aud, sub,
// iat and exp the caller left out. Passing one explicitly overrides the
// default, including to a deliberately wrong value.
func (p *OIDCProvider) Token(claims map[string]any) (string, error) {
	return p.sign(jose.ES256, p.key, oidcKeyID, "at+jwt", claims)
}

// TokenWithType signs claims with an explicit JOSE `typ` header, for proving
// that the RFC 9068 check accepts and rejects the right values.
func (p *OIDCProvider) TokenWithType(typ string, claims map[string]any) (string, error) {
	return p.sign(jose.ES256, p.key, oidcKeyID, typ, claims)
}

// RSAToken signs with RS256 rather than ES256, so a test can show the verifier
// accepts more than one asymmetric algorithm.
func (p *OIDCProvider) RSAToken(claims map[string]any) (string, error) {
	p.rsaWanted.Store(true)

	return p.sign(jose.RS256, p.rsaOnce(), oidcKeyID+"-rsa", "at+jwt", claims)
}

// TokenSignedWith mints a token the provider's key set cannot vouch for, so a
// test can prove the verifier rejects it rather than trusting its contents.
func (p *OIDCProvider) TokenSignedWith(
	alg jose.SignatureAlgorithm, key any, keyID string, claims map[string]any,
) (string, error) {
	return p.sign(alg, key, keyID, "at+jwt", claims)
}

// UnsignedToken produces an `alg: none` token, the oldest JWT bypass there is.
//
// It is assembled by hand because go-jose refuses to sign one, which is itself
// the library behaving correctly.
func (p *OIDCProvider) UnsignedToken(claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]any{"alg": "none", "typ": "at+jwt"})
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(p.withDefaults(claims))
	if err != nil {
		return "", err
	}

	return encodeSegment(header) + "." + encodeSegment(payload) + ".", nil
}

func (p *OIDCProvider) sign(
	alg jose.SignatureAlgorithm, key any, keyID, typ string, claims map[string]any,
) (string, error) {
	opts := (&jose.SignerOptions{}).WithType(jose.ContentType(typ)).WithHeader("kid", keyID)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: key}, opts)
	if err != nil {
		return "", fmt.Errorf("failed to build a signer: %w", err)
	}

	payload, err := json.Marshal(p.withDefaults(claims))
	if err != nil {
		return "", fmt.Errorf("failed to encode the claims: %w", err)
	}

	signed, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("failed to sign the token: %w", err)
	}

	return signed.CompactSerialize()
}

// withDefaults fills the registered claims a verifier requires, leaving any the
// caller set alone so a test can supply a wrong one on purpose.
func (p *OIDCProvider) withDefaults(claims map[string]any) map[string]any {
	now := time.Now()

	defaults := map[string]any{
		"iss": p.Issuer,
		"aud": p.Audience,
		"sub": "test-subject",
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}

	// A new map, so repeated calls cannot accumulate one another's claims.
	out := make(map[string]any, len(defaults)+len(claims))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range claims {
		out[k] = v
	}

	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		// A provider that silently served a truncated document would surface
		// as an unexplained verification failure in whichever test hit it.
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

// encodeSegment renders one JWS segment, which is base64url with the padding
// stripped rather than standard base64.
func encodeSegment(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}
