#!/usr/bin/env python3
"""Ratchet the UI's raw fetch() calls down to zero outside ui/src/api/.

The api client attaches the X-CSRF-Token that every mutating route requires. A
raw fetch() does not, so the middleware answers 403 "CSRF token required" and
the action silently fails. Proved in the real logged-in app (seed#2389):

    raw fetch  PUT /api/v1/settings -> 403 {"error":"CSRF token required"}
    with token PUT /api/v1/settings -> 200 {"status":"updated"}

A read-only raw fetch is a lesser defect but still one: it bypasses the client's
401 -> refresh -> retry, so an expired access token surfaces as an error instead
of being refreshed.

This is a RATCHET against scripts/ui-fetch-baseline.txt, the same shape as the
route-consumer gate: the existing sites are listed with their classification and
must shrink, a new one fails immediately, and an entry that no longer matches
must be removed so the file cannot drift into fiction.
"""

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
BASELINE = ROOT / "scripts" / "ui-fetch-baseline.txt"
SRC = ROOT / "ui" / "src"

# ui/src/api owns the client itself; tests and mocks may call fetch freely.
EXEMPT_DIRS = ("api",)
MUTATING = ("POST", "PUT", "PATCH", "DELETE")


def sites():
    """Yield (relative path, line, method) for each raw fetch() call."""
    for path in sorted(SRC.rglob("*.ts")) + sorted(SRC.rglob("*.tsx")):
        rel = path.relative_to(SRC)
        if rel.parts[0] in EXEMPT_DIRS or ".test." in path.name or "/test/" in str(rel):
            continue
        text = path.read_text(encoding="utf-8")
        for match in re.finditer(r"\bfetch\(", text):
            # The method lives in the options object; 340 chars covers it.
            window = text[match.start() : match.start() + 340]
            found = re.search(r"method:\s*['\"](\w+)['\"]", window)
            method = found.group(1).upper() if found else "GET"
            line = text.count("\n", 0, match.start()) + 1
            yield f"ui/src/{rel}", line, method


def main() -> int:
    found = {f"{path}:{line}": method for path, line, method in sites()}

    baseline = {}
    if BASELINE.exists():
        for raw in BASELINE.read_text(encoding="utf-8").splitlines():
            line = raw.split("#", 1)[0].strip()
            if line:
                baseline[line] = raw

    new = sorted(set(found) - set(baseline))
    gone = sorted(set(baseline) - set(found))

    failed = False
    if new:
        failed = True
        print("::error::raw fetch() outside ui/src/api — use the api client, which "
              "attaches the CSRF token a mutating route requires:")
        for site in new:
            print(f"  {found[site]:6} {site}")
    if gone:
        failed = True
        print(f"::error::baseline entries that no longer exist — remove them from {BASELINE}:")
        for site in gone:
            print(f"  {site}")

    mutating = sum(1 for method in found.values() if method in MUTATING)
    print(
        f"UI fetch gate: {len(found)} raw fetch sites ({mutating} mutating), "
        f"{len(baseline)} baselined."
    )

    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
