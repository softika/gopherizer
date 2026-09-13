package authx

import "context"

// contextKey keeps the identity out of reach of every other package's context
// values. An unexported struct type cannot collide; a string constant can, and
// the collision would be silent.
type contextKey struct{}

var identityKey contextKey

// NewContext returns a copy of ctx carrying id.
func NewContext(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// FromContext returns the Identity carried by ctx.
//
// The boolean separates an unauthenticated request from one that authenticated
// as a principal holding no scopes and no roles. A handler that conflates the
// two grants anonymous callers whatever it meant to grant the empty ones.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}
