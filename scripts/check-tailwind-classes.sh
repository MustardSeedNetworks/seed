#!/bin/sh
# Fails on a Tailwind class Tailwind will never generate.
#
# #2297 substituted family classes for raw utilities and turned `gap-1.5` into
# `gap-tight.5` in eleven places. Nothing caught it: it is valid TypeScript,
# valid JSX, and a string as far as Biome is concerned. Tailwind simply emits no
# rule for it, so the gap became zero and the only symptom was a slightly
# wrong-looking layout.
#
# The shape to catch is a family class carrying a numeric suffix -- gap-tight.5,
# mt-inline.5 -- which is always a bad find-and-replace rather than a real class.
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"

pattern='\b(gap|pad|stack|mb|mt|ml|mr|pt|pb|pl|pr|px|py)-[a-z]+\.[0-9]'

hits=$(grep -rnE "$pattern" ui/src 2>/dev/null || true)

if [ -z "$hits" ]; then
  printf 'No malformed Tailwind classes.\n'
  exit 0
fi

printf '::error::Tailwind classes that will never be generated:\n' >&2
printf '%s\n' "$hits" >&2
printf '\nA family class with a numeric suffix (gap-tight.5) is a bad substitution.\n' >&2
printf 'The fractional step has no family class -- use the raw utility (gap-1.5).\n' >&2
exit 1
