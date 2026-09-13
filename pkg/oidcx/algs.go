package oidcx

import "github.com/coreos/go-oidc/v3/oidc"

// supportedSigningAlgs pins verification to asymmetric signatures.
//
// It is a constant rather than a setting on purpose. `alg: none` and HMAC key
// confusion are the two classic JWT verification bypasses, and a configurable
// list is a lever that can only ever be pulled towards accepting them -- nobody
// widens an algorithm list to strengthen a deployment.
//
// Stating it is also required for correctness, not only for safety: go-oidc's
// default when the field is empty is RS256 alone, which would reject every
// provider that signs with ES256.
var supportedSigningAlgs = []string{
	oidc.RS256, oidc.RS384, oidc.RS512,
	oidc.PS256, oidc.PS384, oidc.PS512,
	oidc.ES256, oidc.ES384, oidc.ES512,
	oidc.EdDSA,
}
