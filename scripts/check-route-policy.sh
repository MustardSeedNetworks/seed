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
