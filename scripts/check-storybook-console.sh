#!/usr/bin/env bash
# check-storybook-console.sh — fail the Storybook run on unexpected error output.
#
# The interaction/a11y job used to pass while logging 60 application error and
# warning lines, every one of them the same thing: Storybook has no daemon, so
# provider bootstrap calls hit the dev server's HTML fallback and React Query
# tried to parse `<!doctype` as JSON. A real regression was indistinguishable
# from that noise -- which is exactly how #2201 hid, a genuine crash in
# VulnerabilityDetailsModal that was unreachable only because parsing failed
# before the data existed.
#
# The msw handlers in ui/.storybook/msw silence the noise. This makes it stay
# silenced: once the log is clean, any new error line is a signal.
#
# Reads the run's output on stdin, or from a file given as $1.

set -euo pipefail

log="${1:-/dev/stdin}"
captured=$(mktemp)
trap 'rm -f "$captured"' EXIT
cat "$log" >"$captured"

# Stories whose PURPOSE is an error state. Each entry is a fixed string matched
# against the log line, with the story that produces it named -- an allow-list
# nobody can explain is a mute button.
#
# Keep this list short. A new entry means either a new deliberate-error story,
# or a real error somebody chose not to fix.
allowed=(
  # src/components/ErrorBoundary.stories.tsx renders a component that throws on
  # purpose, to show the boundary catching it. Both lines below come from that
  # one story: the throw itself, and the boundary reporting what it caught.
  "Storybook induced crash"
  "ErrorBoundary caught an error"
)

# The logger routes every non-error level through console.warn, because the
# lint rules ban console.log and console.debug (src/lib/logger.ts). So the
# console method says nothing about severity, and this greps the level tag the
# logger itself writes.
#
# [WARN] is fatal alongside [ERROR]. It has to be: the noise this replaces was
# half warnings -- "Could not load backend defaults, using hardcoded" is a
# WARN, and it is exactly the signal that an endpoint lost its handler. A gate
# that only watched [ERROR] passed a run with sixteen of them, which is how
# this was caught. [INFO] and [DEBUG] stay allowed; they are informational and
# only reach console.warn because of the lint rule above.
unexpected=$(grep -E '\[(ERROR|WARN)\]' "$captured" || true)

for pattern in "${allowed[@]}"; do
  unexpected=$(grep -Fv -- "$pattern" <<<"$unexpected" || true)
done

# `Error: <message>` lines are raw throws that never reached the logger.
raw=$(grep -E '^\s*.*\bError: ' "$captured" || true)
for pattern in "${allowed[@]}"; do
  raw=$(grep -Fv -- "$pattern" <<<"$raw" || true)
done

if [ -n "$unexpected" ] || [ -n "$raw" ]; then
  echo "::error::Storybook logged application errors. Either the story hit an" >&2
  echo "endpoint no msw handler covers (add one in ui/.storybook/msw/handlers.ts," >&2
  echo "with the shape from the Go handler), or a component genuinely broke." >&2
  echo >&2
  [ -n "$unexpected" ] && printf '%s\n' "$unexpected" >&2
  [ -n "$raw" ] && printf '%s\n' "$raw" >&2
  exit 1
fi

echo "Storybook console clean: no unexpected error output"
