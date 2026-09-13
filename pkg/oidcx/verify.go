package oidcx

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/pkg/authx"
	"github.com/softika/gopherizer/pkg/errorx"
)

// idTokenOnlyClaims are defined by OpenID Connect Core as ID token claims.
//
// An access token carrying one is an ID token being replayed in its place, and
// this check holds even when the audience has been misconfigured to an OAuth
// client id -- the mistake that makes the audience check alone insufficient,
// since an ID token's `aud` is exactly that client id.
//
// at_hash and c_hash carry no false-positive risk: no provider emits them in an
// access token. nonce has one, Keycloak before version 17, which stopped in
// 2022; the failure there is a refused token rather than an accepted one.
var idTokenOnlyClaims = []string{"at_hash", "c_hash", "nonce"}

// accessTokenTypes are the `typ` header values RFC 9068 defines for a JWT
// access token. Media type suffixes are compared case-insensitively.
var accessTokenTypes = []string{"at+jwt", "application/at+jwt"}

type verifier struct {
	cfg    config.OIDCConfig
	tokens *oidc.IDTokenVerifier
}

// Verify checks the token's signature, issuer, audience and expiry, then
// resolves its claims to an identity.
//
// Rejections are deliberately uniform: the caller learns that the credential
// was refused and never which check refused it, because naming the failed check
// tells an attacker which part they already have right. The detail goes to the
// log instead, where the api layer records it against the request id.
func (v *verifier) Verify(ctx context.Context, rawToken string) (authx.Identity, error) {
	token, err := v.tokens.Verify(ctx, rawToken)
	if err != nil {
		return authx.Identity{}, errorx.NewError(
			fmt.Errorf("failed to verify the bearer token: %w", err),
			classify(err),
		)
	}

	var claims map[string]any
	if err = token.Claims(&claims); err != nil {
		return authx.Identity{}, rejected(fmt.Errorf("failed to read the token claims: %w", err))
	}

	for _, claim := range idTokenOnlyClaims {
		if _, found := claims[claim]; found {
			return authx.Identity{}, rejected(
				fmt.Errorf("token carries %q, which marks it an id token rather than an access token", claim),
			)
		}
	}

	if v.cfg.RequireTypedAccessToken {
		if err = requireAccessTokenType(rawToken); err != nil {
			return authx.Identity{}, rejected(err)
		}
	}

	// A token nothing can be attributed to is not an identity, and every later
	// decision -- authorization, audit, ownership -- reads the subject.
	if token.Subject == "" {
		return authx.Identity{}, rejected(errors.New("token carries no subject"))
	}

	return authx.Identity{
		Subject: token.Subject,
		Scopes:  claimList(claims, v.cfg.ScopeClaim),
		Roles:   claimList(claims, v.cfg.RolesClaim),
		Claims:  claims,
	}, nil
}

// requireAccessTokenType enforces RFC 9068's `typ` header.
//
// The header is read back off the raw token because go-oidc does not expose it.
// That is only sound because this runs after verification succeeded: the
// signature covers the header, so what is decoded here is what the provider
// signed rather than something the caller chose.
func requireAccessTokenType(rawToken string) error {
	encoded, _, found := strings.Cut(rawToken, ".")
	if !found {
		return errors.New("token is not a signed jwt")
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("failed to decode the token header: %w", err)
	}

	var header struct {
		Typ string `json:"typ"`
	}
	if err = json.Unmarshal(raw, &header); err != nil {
		return fmt.Errorf("failed to parse the token header: %w", err)
	}

	if !slices.ContainsFunc(accessTokenTypes, func(candidate string) bool {
		return strings.EqualFold(candidate, header.Typ)
	}) {
		return fmt.Errorf("token type %q is not an access token", header.Typ)
	}

	return nil
}

// transportFailures are the messages go-oidc emits when it could not obtain the
// key set, either because the endpoint was unreachable or because it answered
// with something other than a key set.
//
// Matching on message text is unpleasant, and it is forced. go-oidc joins the
// transport failure into its result with %v rather than %w
// (oidc/verify.go: "failed to verify signature: %v"), which flattens
// everything beneath it into a string -- so errors.As cannot reach the
// net.Error that is plainly visible in the text. A non-2xx key response is
// worse still: oidc/jwks.go builds it with no wrapped error at all.
//
// "oidc: get keys failed" covers both cases; "fetching keys" is the layer
// above it, kept so a change to either one alone does not silently reclassify
// an outage as a rejected credential.
var transportFailures = []string{
	"oidc: get keys failed",
	"fetching keys",
}

// classify separates a credential that was refused from one that could not be
// checked at all.
//
// go-oidc reports both as a failure to verify, but they mean opposite things:
// an unreachable key endpoint is the provider's outage, and answering 401 for
// it makes every caller appear to present bad credentials at the same instant
// -- which points an investigation at the clients rather than at the provider.
func classify(err error) errorx.ErrorType {
	// Kept ahead of the text matching, so this starts working properly the day
	// go-oidc wraps with %w.
	var netErr net.Error
	if errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded) {
		return errorx.ErrUnavailable
	}

	message := err.Error()
	for _, marker := range transportFailures {
		if strings.Contains(message, marker) {
			return errorx.ErrUnavailable
		}
	}

	return errorx.ErrUnauthorized
}

func rejected(err error) error {
	return errorx.NewError(err, errorx.ErrUnauthorized)
}
