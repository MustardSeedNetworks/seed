#!/usr/bin/env bash
# check-filename-policy.sh — capability-package file-naming gate.
#
# The flat internal/api package uses filename prefixes (handlers_, jobs_) to
# separate concerns within one large package. Those prefixes are meaningful
# *only* there. Once code is decomposed into a dedicated capability package
# (internal/<cap>/), the package declaration already supplies that context, so
# the prefix becomes stutter: internal/health/handlers_health_api.go re-states
# "health" twice and re-asserts a layer fact the package name already makes.
#
# Best practice in a decomposed package is to name files for their role *within*
# the package (health.go, checks.go, handler.go) — never by re-stating the layer
# (handlers_/jobs_) or the capability the package itself already represents.
# This is not a Go *language* rule (the toolchain accepts these names); it is an
# architecture-consistency rule that keeps the internal/api strangle honest:
# monolith vocabulary stays in the monolith and is dropped on the way out.
#
# This gate fails if any handlers_*.go / jobs_*.go file exists outside
# internal/api, UNLESS the file's own package directory is named for that token
# (e.g. internal/platform/jobs/jobs.go is the eponymous file of package `jobs`,
# and jobs_test.go is its standard test — both idiomatic, not monolith baggage).
# The gate is green on the current tree and only fires when a decomposition
# carries the monolith's grouping prefix forward instead of dropping it.
#
# Run locally: scripts/check-filename-policy.sh
set -euo pipefail

violations=""
while IFS= read -r f; do
	[[ -z "$f" ]] && continue
	dir=$(basename "$(dirname "$f")")  # parent package directory
	prefix="${f##*/}"; prefix="${prefix%%_*}"  # handlers | jobs
	# Allow the eponymous package file/test (dir named for the prefix token).
	[[ "$dir" == "$prefix" ]] && continue
	violations+="$f"$'\n'
done < <(find . \
	-type f \
	\( -name 'handlers_*.go' -o -name 'jobs_*.go' \) \
	-not -path './vendor/*' \
	-not -path './internal/api/*')
violations=$(printf '%s' "$violations" | sed '/^[[:space:]]*$/d' | sort)

if [[ -n "$violations" ]]; then
	echo "❌ Filename-policy gate: monolith file-naming prefixes (handlers_/jobs_)"
	echo "   must not appear outside internal/api. A decomposed capability package"
	echo "   names files for their role within the package — drop the prefix:"
	echo "     internal/health/handlers_health_api.go  →  internal/health/health.go"
	echo ""
	echo "$violations"
	exit 1
fi

echo "✓ Filename-policy gate: no monolith naming prefixes outside internal/api."
