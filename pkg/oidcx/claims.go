package oidcx

import (
	"slices"
	"strings"
)

// lookupClaim resolves a claim by name, falling back to a dotted path.
//
// The whole name is tried as a top-level key first, so a namespaced claim that
// itself contains dots -- "https://example.com/roles", which Auth0 issues --
// resolves before the path walk can misread it as a path. Only then is the name
// split, which is how Keycloak's nested "realm_access.roles" is reached.
//
// One function therefore covers all three conventions in use, which is what
// keeps the claim names configurable rather than hard-coded per provider.
func lookupClaim(claims map[string]any, name string) (any, bool) {
	if name == "" {
		return nil, false
	}

	if value, ok := claims[name]; ok {
		return value, true
	}

	var current any = claims
	for _, segment := range strings.Split(name, ".") {
		nested, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		if current, ok = nested[segment]; !ok {
			return nil, false
		}
	}

	return current, true
}

// claimList reads a claim that providers spell either as a space-delimited
// string or as an array.
//
// OAuth2 defines `scope` as the former and Entra ID's `scp` follows it, while
// `roles` and `groups` are almost always the latter -- and a deployment can
// point either setting at either claim, so both shapes have to be accepted
// wherever either is read.
func claimList(claims map[string]any, name string) []string {
	value, ok := lookupClaim(claims, name)
	if !ok {
		return nil
	}

	switch typed := value.(type) {
	case string:
		return strings.Fields(typed)

	case []string:
		return slices.Clone(typed)

	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}

		return out

	default:
		return nil
	}
}
