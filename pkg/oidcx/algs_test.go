package oidcx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The pinned list is the only thing standing between the verifier and the two
// classic JWT bypasses, so widening it must take a deliberate edit that fails
// this test rather than a quiet addition during a debugging session.
//
// That the checks themselves work is proven behaviourally in oidcx_test.go,
// where an alg:none token and an HMAC-signed token are both rejected end to
// end. This guards the declaration those rejections depend on.
func TestSupportedSigningAlgsAreAsymmetric(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, supportedSigningAlgs,
		"an empty list is not permissive but wrong: go-oidc falls back to RS256 alone")

	for _, alg := range supportedSigningAlgs {
		assert.NotEqual(t, "none", strings.ToLower(alg), "alg:none must never be accepted")
		assert.False(t, strings.HasPrefix(alg, "HS"),
			"%s is symmetric: the verifying key would also be a signing key", alg)
	}
}
