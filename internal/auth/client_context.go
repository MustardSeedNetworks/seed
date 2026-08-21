// SPDX-License-Identifier: BUSL-1.1

package auth

// client_context.go carries the authenticated session's owning client through
// the request context.
//
// Why not a request header, which is how this codebase already threads the
// username and the PAT scope: a header is writable by the caller. Those two
// are overwritten by the middleware on every authenticated path, so forging
// them fails — but tenancy is the one identity that must never be capable of
// arriving from the request at all, on any path, including ones that bypass
// auth. The context value has no wire representation, so there is nothing for
// a caller to set.

import (
	"context"
	"errors"
)

// contextKey is the unexported key type for this package's context values, so
// nothing outside it can construct a key that collides.
type contextKey string

// clientIDKey addresses the session's owning client.
const clientIDKey contextKey = "client_id"

// ErrNoClientID is returned when a request context carries no client claim.
// Callers must treat it as a denial, never as "use the default client" —
// manufacturing a tenant here would reintroduce exactly the hole the claim
// closes.
var ErrNoClientID = errors.New("auth: request carries no client claim")

// DefaultClientID is the id of the seeded default client, duplicated here
// because internal/auth cannot import internal/database without a cycle.
// It is the client minted for the storeless (config-file user) path, which
// #1799 removes. TestDefaultClientIDMatchesDatabase pins the two together.
const DefaultClientID = "default"

// WithClientID returns a context carrying the session's owning client.
func WithClientID(ctx context.Context, clientID string) context.Context {
	return context.WithValue(ctx, clientIDKey, clientID)
}

// ClientIDFromContext returns the session's owning client, or ErrNoClientID
// when the request was not authenticated through a path that mints one.
func ClientIDFromContext(ctx context.Context) (string, error) {
	clientID, ok := ctx.Value(clientIDKey).(string)
	if !ok || clientID == "" {
		return "", ErrNoClientID
	}
	return clientID, nil
}
