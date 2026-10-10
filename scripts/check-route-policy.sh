#!/usr/bin/env bash
# check-route-policy.sh — capability-registry enforcement gate (ADR-0002).
#
# Every route goes through foundation's route.Registrar, which composes each
# route's policy (limiter, auth, method gate, CSRF, feature, scope, body cap) in
# one canonical order. Hand-wrapping a route on a mux of our own is how a
# mutating route silently ships without its protection. The rule is
# foundation's, run from the pinned module copy so the gate and the registrar
# it enforces move on one version.
#
# Run locally: scripts/check-route-policy.sh
set -euo pipefail

route_pkg_dir=$(go list -f '{{.Dir}}' github.com/MustardSeedNetworks/foundation/pkg/httpserver/route)
bash "$route_pkg_dir/check-route-policy.sh" internal/api

# Second rule (#2632): a route carrying a role gate (Scope) or a licence gate
# (Feature) must declare Auth. Both gates resolve the caller from an identity
# only the auth middleware establishes, so a gated route without it gates on
# what the caller supplied — which is how a viewer could rewrite the OAuth
# provider config.
#
# Run as a Go test over the real route table rather than a grep: the table
# builds paths from constants, which a grep over the literals cannot resolve.
if ! go test ./internal/api/ -run TestNoGatedRouteBypassesAuth -count=1 >/dev/null; then
	echo "❌ Route-policy gate (#2632): a role- or feature-gated route does not"
	echo "   declare Auth. Re-run for the route names:"
	echo "   go test ./internal/api/ -run TestNoGatedRouteBypassesAuth -count=1"
	exit 1
fi

echo "✓ Route-policy gate: every role- or feature-gated route declares Auth."
