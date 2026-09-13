package authx

import (
	"strings"
)

// Requirement is one conjunction: every scope and every role named in it must
// be present on the identity for it to be satisfied.
type Requirement struct {
	Scopes []string
	Roles  []string
}

// Requirements is a disjunction of Requirement -- satisfying any single one
// authorizes the caller.
//
// Disjunctive normal form is deliberate. It is the shape OpenAPI's own security
// field already takes, so OpenAPI below is a rename rather than a normalisation
// pass, and Satisfied is two nested loops rather than a tree walk.
//
// The zero value authorizes nobody. It is not a way to spell "open": whether an
// operation is guarded at all is decided by whether it is passed to api.secured,
// never by what a requirement evaluates to. Keeping those separate is what stops
// an expression that happens to reduce to empty -- AllOf over a term read from
// configuration, say -- from turning a rule nobody can satisfy into a public
// endpoint.
type Requirements []Requirement

// Scope requires every named scope.
func Scope(names ...string) Requirements {
	return Requirements{{Scopes: names}}
}

// Role requires every named role.
func Role(names ...string) Requirements {
	return Requirements{{Roles: names}}
}

// Authenticated is satisfied by any verified token.
//
// It is the weakest requirement that still demands one -- and, naming neither a
// scope nor a role, the only requirement an ID token replayed as an access
// token could satisfy. Prefer a scope wherever there is one to name.
//
// A function rather than a package variable, so a caller cannot mutate the
// shared value out from under every other use of it.
func Authenticated() Requirements {
	return Requirements{{}}
}

// AnyOf is satisfied when any one of rs is.
func AnyOf(rs ...Requirements) Requirements {
	var out Requirements
	for _, r := range rs {
		out = append(out, r...)
	}

	return out
}

// AllOf is satisfied only when every one of rs is.
//
// Each argument is itself a disjunction, so combining them is a cross product:
// one alternative for every way of picking a term from each argument. Two
// two-term arguments therefore produce four alternatives, which is why this
// stays cheap only for the small requirements an operation actually declares.
//
// An argument that nobody can satisfy makes the whole conjunction
// unsatisfiable, and is reported as such rather than skipped.
func AllOf(rs ...Requirements) Requirements {
	// One empty conjunction is the identity element: it constrains nothing, so
	// crossing it with the first argument yields that argument unchanged.
	out := Requirements{{}}

	for _, r := range rs {
		if len(r) == 0 {
			return nil
		}

		next := make(Requirements, 0, len(out)*len(r))
		for _, acc := range out {
			for _, term := range r {
				next = append(next, Requirement{
					Scopes: concat(acc.Scopes, term.Scopes),
					Roles:  concat(acc.Roles, term.Roles),
				})
			}
		}
		out = next
	}

	return out
}

// Satisfied reports whether id meets at least one alternative.
func (r Requirements) Satisfied(id Identity) bool {
	for _, alt := range r {
		if alt.satisfied(id) {
			return true
		}
	}

	return false
}

func (r Requirement) satisfied(id Identity) bool {
	for _, scope := range r.Scopes {
		if !id.HasScope(scope) {
			return false
		}
	}

	for _, role := range r.Roles {
		if !id.HasRole(role) {
			return false
		}
	}

	return true
}

// OpenAPI projects the requirement onto an OpenAPI security field.
//
// The projection is lossy by construction: OpenAPI security requirements can
// express scopes and cannot express roles, so a role-only alternative documents
// as "a token is required" while the role itself stays enforced in code only.
//
// That loss is safe because the projection runs one way, from the value that is
// actually enforced. The document can therefore only ever understate what is
// enforced -- never overstate it, which is the direction that would mislead.
func (r Requirements) OpenAPI(scheme string) []map[string][]string {
	if len(r) == 0 {
		return nil
	}

	out := make([]map[string][]string, 0, len(r))
	for _, alt := range r {
		// An explicitly empty list, never nil: OpenAPI distinguishes "no
		// scopes required" from a null, and huma serializes nil as the latter.
		scopes := make([]string, 0, len(alt.Scopes))
		scopes = append(scopes, alt.Scopes...)

		out = append(out, map[string][]string{scheme: scopes})
	}

	return out
}

// String renders the requirement for a denial log line.
func (r Requirements) String() string {
	if len(r) == 0 {
		return "none satisfiable"
	}

	alts := make([]string, 0, len(r))
	for _, alt := range r {
		var parts []string
		for _, scope := range alt.Scopes {
			parts = append(parts, "scope:"+scope)
		}
		for _, role := range alt.Roles {
			parts = append(parts, "role:"+role)
		}

		if len(parts) == 0 {
			alts = append(alts, "authenticated")
			continue
		}
		alts = append(alts, strings.Join(parts, "+"))
	}

	return strings.Join(alts, " or ")
}

// concat returns a new slice holding a then b, so neither input is aliased into
// the result. Requirements are values that get combined and reused; sharing a
// backing array between them would let one alternative rewrite another.
func concat(a, b []string) []string {
	if len(a)+len(b) == 0 {
		return nil
	}

	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)

	return append(out, b...)
}
