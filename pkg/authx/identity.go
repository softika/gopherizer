// Package authx models an authenticated caller and the authorization rules a
// service writes against one.
//
// It knows nothing about OIDC, JWTs or HTTP: an Identity is produced by
// whatever authenticated the request, and a Requirement is evaluated against
// it. That separation is what lets the authorization vocabulary survive a
// change of identity provider, and it is why this package has no dependencies.
package authx

import "slices"

// Identity is a caller whose credential has already been verified.
//
// Constructing one is an assertion that verification succeeded, so nothing
// downstream re-checks it. Only the package that verifies credentials should
// build these.
type Identity struct {
	// Subject is the principal the credential names. An identity without one
	// is not an identity, so the verifier rejects a token with an empty `sub`
	// rather than producing an Identity nothing can attribute.
	Subject string
	Scopes  []string
	Roles   []string
	// Claims are the remaining verified claims, for a service that needs one
	// the template does not model -- a tenant id, an email. Reading from here
	// is reading untyped shape: assert, do not convert.
	Claims map[string]any
}

// HasScope reports whether the identity carries scope.
func (i Identity) HasScope(scope string) bool {
	return slices.Contains(i.Scopes, scope)
}

// HasRole reports whether the identity carries role.
func (i Identity) HasRole(role string) bool {
	return slices.Contains(i.Roles, role)
}
