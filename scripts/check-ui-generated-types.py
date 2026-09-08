#!/usr/bin/env python3
"""Ratchet hand-typed UI interfaces that shadow a generated wire type down to zero.

`ui/src/types/generated/` is produced from the JSON Schemas in docs/schemas/api/,
which are produced from the Go DTOs. A hand-typed `interface DiscoveredDevice`
elsewhere in ui/src does not merely duplicate that type — it is usually a
*subset* of it, and a field the local type omits is a field the code cannot
name, so the capability behind it is unreachable rather than merely awkward.

This is the class that caused seed#2391: `useAuth` declared its own
`LoginResponse` carrying only `token` and `expires`, so `data.mfaRequired` would
not compile, the login form could never implement a second factor, and enrolling
TOTP locked the account permanently. Nothing pointed at the type.

The gate matches on NAME, which catches the shadowing case and not a hand-typed
interface under a different name — the harder half, still open (seed#2393). Name
matching is worth having on its own: every one of the shadows found in that
triage would have been caught by it.

Entries are keyed on `path:InterfaceName`, deliberately NOT on a line number:
scripts/ui-fetch-baseline.txt is line-anchored and an unrelated edit that moves
a baselined site reads as one new site plus one stale entry (seed#2513).

A baselined entry must carry a `# reason` saying why the local declaration is
deliberate — a narrower view, a different wire shape that merely collides by
name, or a UI-only type that is not a DTO at all.
"""

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
BASELINE = ROOT / "scripts" / "ui-generated-type-baseline.txt"

INTERFACE = re.compile(r"^(?:export )?interface (\w+)", re.M)


def generated_names(src):
    """Every interface name the generated wire types export."""
    names = set()
    for path in sorted((src / "types" / "generated").rglob("*.ts")):
        names.update(INTERFACE.findall(path.read_text(encoding="utf-8")))
    return names


def shadows(src):
    """Yield `path:Name` for each hand-typed interface shadowing a generated one."""
    generated = src / "types" / "generated"
    names = generated_names(src)
    for path in sorted(src.rglob("*.ts")) + sorted(src.rglob("*.tsx")):
        if generated in path.parents:
            continue
        if ".test." in path.name or ".stories." in path.name:
            continue
        rel = path.relative_to(src)
        if rel.parts[0] == "test":
            continue
        for name in INTERFACE.findall(path.read_text(encoding="utf-8")):
            if name in names:
                yield f"ui/src/{rel}:{name}"


def run(root, baseline_path, out=sys.stdout) -> int:
    found = set(shadows(root / "ui" / "src"))

    baseline = {}
    if baseline_path.exists():
        for raw in baseline_path.read_text(encoding="utf-8").splitlines():
            entry = raw.split("#", 1)[0].strip()
            if entry:
                baseline[entry] = raw

    new = sorted(found - set(baseline))
    gone = sorted(set(baseline) - found)
    unexplained = sorted(entry for entry, raw in baseline.items() if "#" not in raw)

    failed = False
    if new:
        failed = True
        print(
            "::error::hand-typed interface shadows a generated wire type — import it "
            f"from ui/src/types/generated, or baseline it in {baseline_path} with a reason:",
            file=out,
        )
        for entry in new:
            print(f"  {entry}", file=out)
    if gone:
        failed = True
        print(
            f"::error::baseline entries that no longer exist — remove them from {baseline_path}:",
            file=out,
        )
        for entry in gone:
            print(f"  {entry}", file=out)
    if unexplained:
        failed = True
        print(
            "::error::baseline entries with no `# reason` — say why the local type is deliberate:",
            file=out,
        )
        for entry in unexplained:
            print(f"  {entry}", file=out)

    print(f"UI generated-type gate: {len(found)} shadows, {len(baseline)} baselined.", file=out)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(run(ROOT, BASELINE))
