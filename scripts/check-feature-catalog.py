#!/usr/bin/env python3
"""check-feature-catalog.py — license catalog ⇄ code gate.

Every string in starterFeatures() and proFeatures() in
internal/license/policy.go is a feature a Starter or Pro customer is paying
for. On 2026-07-10 four such strings were found with no backing code and the
removal commit (#1753) claimed to delete them — the strings are still in the
list. The 2026-09-02 audit found 17 of 29 with no gating reference anywhere.
Nothing stopped either.

A sold feature is BACKED when its string appears in Go outside
internal/license and outside tests: a `feature: "x"` registry row, a
`HasFeature("x")` check, or any other quoted use. Unbacked features are a
RATCHET against scripts/feature-catalog-baseline.txt: growth fails, and a
baselined feature that has since been backed or removed from the catalog
fails too, so the baseline is the S1-3 work queue rather than an allow-list.

Run locally: scripts/check-feature-catalog.py   ·   regenerate: --update
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

POLICY = "internal/license/policy.go"
BASELINE = "scripts/feature-catalog-baseline.txt"
CATALOG_FUNCS = re.compile(r"func (starterFeatures|proFeatures)\(\) \[\]string \{(.*?)\n\}", re.S)
FEATURE_STRING = re.compile(r'"([a-z][a-z0-9_]*)"')


def catalog(root: Path) -> set[str]:
    text = (root / POLICY).read_text(encoding="utf-8")
    features: set[str] = set()
    for _, body in CATALOG_FUNCS.findall(text):
        body = re.sub(r"//[^\n]*", "", body)
        features.update(FEATURE_STRING.findall(body))
    return features


def backed(root: Path, features: set[str]) -> dict[str, list[str]]:
    refs: dict[str, list[str]] = {f: [] for f in features}
    for file in sorted(root.rglob("*.go")):
        rel = file.relative_to(root).as_posix()
        if rel.startswith("internal/license/") or file.name.endswith("_test.go") or "/vendor/" in rel:
            continue
        text = file.read_text(encoding="utf-8")
        for feature in features:
            if f'"{feature}"' in text:
                refs[feature].append(rel)
    return refs


def read_baseline(path: Path) -> dict[str, str]:
    entries: dict[str, str] = {}
    if path.exists():
        for raw in path.read_text(encoding="utf-8").splitlines():
            line = raw.strip()
            if line and not line.startswith("#"):
                key, _, reason = line.partition("#")
                entries[key.strip()] = reason.strip()
    return entries


def run(root: Path, baseline_path: Path, update: bool = False, out=sys.stdout) -> int:
    features = catalog(root)
    if not features:
        print(f"::error::no catalog strings found in {POLICY} — is the regex stale?", file=out)
        return 2
    refs = backed(root, features)
    unbacked = sorted(f for f, where in refs.items() if not where)
    previous = read_baseline(baseline_path)

    if update:
        lines = ["# Sold features (starterFeatures/proFeatures in policy.go) with no gating reference in Go.",
                 "# Ratchet; see check-feature-catalog.py. One feature per line, optional `# reason`.",
                 "# Remove a line when the feature gains a gate or leaves the catalog."]
        lines += [f"{f}  # {previous[f]}" if previous.get(f) else f for f in unbacked]
        baseline_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
        print(f"Wrote {len(unbacked)} unbacked feature(s) to {baseline_path}", file=out)
        return 0

    failed = False
    new = [f for f in unbacked if f not in previous]
    stale = [f for f in previous if f not in unbacked]
    if new:
        failed = True
        print("::error::sold features with no gating reference in Go that are not in the baseline "
              "(gate them with requireFeature/HasFeature, delete them from policy.go, or baseline with a reason):", file=out)
        for f in new:
            print(f"  {f}", file=out)
    if stale:
        failed = True
        print(f"::error::baseline entries now backed or no longer sold — remove them from {baseline_path}:", file=out)
        for f in stale:
            print(f"  {f}", file=out)
    print(f"Feature-catalog gate: {len(features)} sold features, {len(features) - len(unbacked)} backed, "
          f"{len(unbacked)} unbacked ({len(previous)} baselined).", file=out)
    return 1 if failed else 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--update", action="store_true", help="rewrite the baseline from the current tree")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    args = parser.parse_args()
    return run(args.root, args.root / BASELINE, update=args.update)


if __name__ == "__main__":
    sys.exit(main())
