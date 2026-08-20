#!/usr/bin/env python3
"""TypeScript strictness contract for the MSN family.

The four products keep their own `ui/tsconfig.app.json` — there is no master
repo — so the flag sets drift silently. They did: for a while seed and stem
set `isolatedModules` without `verbatimModuleSyntax`, trellis the reverse, and
niac had `verbatimModuleSyntax: false` written out explicitly. Nothing failed,
because a missing strictness flag never announces itself; it just quietly
checks less than its sibling does.

This asserts the flags this repo has agreed to carry, with the value each must
have. It deliberately does *not* reach into a sibling checkout — a gate that
does only works on a machine that has one. The list below is the contract;
changing it is a decision someone makes on purpose, in four places.

Flags are read with a per-flag match rather than by parsing the file as JSON:
tsconfig is JSONC, and the obvious "strip the comments first" regex eats the
`/*` inside a path glob like `"@/*": ["./src/*"]`. A contract gate that cannot
read three of the four repos is worse than no gate.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

CONFIG = Path(__file__).resolve().parent.parent / "ui" / "tsconfig.app.json"

# Flag -> required value. Keep in step with the sibling repos' copies.
REQUIRED: dict[str, bool] = {
    "strict": True,
    "isolatedModules": True,
    "verbatimModuleSyntax": True,
    "erasableSyntaxOnly": True,
    "noUncheckedIndexedAccess": True,
    "noUnusedLocals": True,
    "noUnusedParameters": True,
    "noFallthroughCasesInSwitch": True,
    "noUncheckedSideEffectImports": True,
}

# Not yet agreed anywhere; listed so its absence reads as pending rather than
# forgotten. See the modernization plan's Step 6.
PENDING: tuple[str, ...] = ("exactOptionalPropertyTypes",)


def read_flag(text: str, flag: str) -> bool | None:
    """Return the flag's value, or None when it is not set at all."""
    match = re.search(rf'"{re.escape(flag)}"\s*:\s*(true|false)', text)
    return None if match is None else match.group(1) == "true"


def uncommented(path: Path) -> str:
    lines = [ln for ln in path.read_text(encoding="utf-8").splitlines() if not ln.lstrip().startswith("//")]
    return "\n".join(lines)


def main() -> int:
    if not CONFIG.is_file():
        print(f"no {CONFIG} - nothing to check")
        return 0

    text = uncommented(CONFIG)
    problems: list[str] = []

    for flag, want in REQUIRED.items():
        value = read_flag(text, flag)
        if value is None:
            problems.append(f"  {flag} is missing - the fleet sets it to {str(want).lower()}")
        elif value != want:
            problems.append(
                f"  {flag} is {str(value).lower()} - the fleet sets it to {str(want).lower()}"
            )

    if problems:
        print("FAIL: ui/tsconfig.app.json has drifted from the fleet's strictness contract:")
        print("\n".join(problems))
        print("\nIf the change is intended, land it in seed, stem, trellis and niac-go,")
        print("then update REQUIRED in each repo's scripts/check-tsconfig-flags.py.")
        return 1

    pending = [flag for flag in PENDING if read_flag(text, flag) is not None]
    if pending:
        print(f"note: {', '.join(pending)} is set here but not yet agreed fleet-wide")

    print(f"OK: all {len(REQUIRED)} agreed strictness flags are set.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
