#!/usr/bin/env python3
"""resolve-token-refs.py — undefined design-token gate.

Every semantic color token referenced as a Tailwind utility (bg-/text-/
border-/ring-/...) MUST resolve to a --color-* defined in index.css. An
undefined token (e.g. bg-status-danger when only --color-status-error exists)
compiles to nothing and silently renders no color, so the palette/hex guard in
check-token-discipline.sh cannot see it. This check closes that gap.

Usage: resolve-token-refs.py <target-dir>
  <target-dir> contains index.css and the *.ts/*.tsx tree (e.g. "src" when run
  from ui/, or "ui/src" from the repo root). Exits non-zero on any undefined
  token; prints file:line for each.
"""
import re
import sys
from pathlib import Path

target = Path(sys.argv[1] if len(sys.argv) > 1 else "ui/src")
css = (target / "index.css").read_text()

# Defined --color-* names from :root / .dark / @theme — the source of truth.
defined = set(re.findall(r"--color-([a-z0-9-]+)\s*:", css))
# Known semantic roots = first segment of each defined token. Only utilities
# whose name starts with one of these roots are treated as token references;
# this excludes built-ins (text-sm, border-2, shadow-lg, bg-transparent, ...).
roots = {name.split("-")[0] for name in defined}

PREFIXES = (
    "bg|text|border|ring|fill|stroke|from|via|to|shadow|outline|"
    "divide|placeholder|caret|decoration|accent"
)
util = re.compile(rf"\b(?:{PREFIXES})-([a-z][a-z0-9]*(?:-[a-z0-9]+)*)")
SKIP = re.compile(r"\.(test|spec|stories|mock)\.(ts|tsx)$|/styles/|/constants/")

undefined: dict[str, list[str]] = {}
for path in target.rglob("*.ts*"):
    if path.suffix not in (".ts", ".tsx") or SKIP.search(str(path)):
        continue
    for i, line in enumerate(path.read_text().splitlines(), 1):
        for name in util.findall(line):
            if name.split("-")[0] in roots and name not in defined:
                undefined.setdefault(name, []).append(f"{path}:{i}")

if not undefined:
    sys.exit(0)

for tok in sorted(undefined):
    locs = undefined[tok]
    print(f"  undefined token: {tok}  ({len(locs)}x)  e.g. {locs[0]}")
    for loc in locs[1:6]:
        print(f"      {loc}")
sys.exit(1)
