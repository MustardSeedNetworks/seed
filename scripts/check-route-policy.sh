#!/usr/bin/env bash
# check-route-policy.sh — capability-registry enforcement gate (ADR-0002).
#
# Every API route MUST be registered through the capability registry
# (register / registerAll in internal/api/route.go), which composes its
# per-route policy — role gate, license-feature gate, rate limiting — in one
# canonical order. Hand-wrapping a route directly on the mux bypasses that
# composition and is how a mutating route silently ships without a role/feature
# gate ("forgot the wrapper", a documented regression class).
#
# This gate fails if any API route (an APIVersionPrefix path) is registered
# directly via s.mux.Handle/HandleFunc instead of through register(). The only
# permitted direct mux registrations are infra introspection / static serving
# (/__version, /__capabilities, "/" SPA fallback) and register()'s own
# implementation — none of which reference APIVersionPrefix.
#
# Run locally: scripts/check-route-policy.sh
set -euo pipefail

API_DIR="internal/api"

violations=$(grep -rnE 's\.mux\.Handle(Func)?\(APIVersionPrefix' "$API_DIR"/*.go \
	| grep -v '_test.go' || true)

if [[ -n "$violations" ]]; then
	echo "❌ Route-policy gate (ADR-0002): API routes must be registered via"
	echo "   register()/registerAll(), not s.mux.Handle*() directly."
	echo "   Replace the direct registration with a route{} entry"
	echo "   (path/handler/minRole/feature/rateLimited)."
	echo ""
	echo "$violations"
	exit 1
fi

echo "✓ Route-policy gate: all API routes go through the capability registry."

# Second rule (#2632): a route carrying a role gate (minRole) or a licence
# gate (feature) must not sit on a path that skips the JWT middleware. Both
# gates resolve the caller from an identity only that middleware establishes,
# so a gated route on a bypass path gates on what the caller supplied — which
# is how a viewer could rewrite the OAuth provider config.
#
# Run as a Go test rather than a grep: the route table builds paths from
# constants (APIVersionPrefix + "/..."), which a grep over the literals cannot
# resolve, and the bypass predicate is a function, not a list.
if ! go test ./internal/api/ -run TestNoGatedRouteBypassesAuth -count=1 >/dev/null; then
	echo "❌ Route-policy gate (#2632): a role- or feature-gated route bypasses"
	echo "   the auth middleware. Re-run for the route names:"
	echo "   go test ./internal/api/ -run TestNoGatedRouteBypassesAuth -count=1"
	exit 1
fi

echo "✓ Route-policy gate: no role- or feature-gated route bypasses auth."
