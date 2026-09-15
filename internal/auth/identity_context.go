// SPDX-License-Identifier: BUSL-1.1

package auth

// identity_context.go carries the authenticated caller's identity — username
// and, for a personal access token, the token's role cap — on the request
// context.
//
// #2632: both used to ride on request headers (X-Username, X-Token-Scope) that
// the middleware set and the role gate read back. A header is writable by the
// caller, so any route that reached a handler without passing through the
// middleware handed the role gate whatever the client had typed. The context
// key is unexported and untyped-collidable, so only this package can set it.

import "context"

// usernameKey addresses the authenticated caller's username.
const usernameKey contextKey = "username"

// tokenScopeKey addresses the role cap of the personal access token that
// authenticated the request, if one did.
const tokenScopeKey contextKey = "token_scope"

// WithUsername returns a context carrying the authenticated caller's username.
// Set only after a credential has been validated.
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

// UsernameFromContext returns the authenticated caller's username, or "" when
// the request did not come through an authenticating middleware. Callers must
// read "" as "no caller" — never as a caller worth looking up.
func UsernameFromContext(ctx context.Context) string {
	username, _ := ctx.Value(usernameKey).(string)
	return username
}

// WithTokenScope returns a context carrying a personal access token's role cap.
func WithTokenScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, tokenScopeKey, scope)
}

// TokenScopeFromContext returns the role cap of the personal access token that
// authenticated the request, or "" when no token did.
func TokenScopeFromContext(ctx context.Context) string {
	scope, _ := ctx.Value(tokenScopeKey).(string)
	return scope
}
