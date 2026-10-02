#!/usr/bin/env python3
"""Hold the JavaScript a first page load downloads under a gzip budget.

The first load fetches the entry script and every chunk index.html
modulepreloads with it. Counting the entry file alone is not enough: the
bundler moves modules shared with a lazy chunk into sibling chunks, so the
entry can shrink while the first load does not. So the gate reads the built
index.html and sums every script it loads.

Vendor chunks (the `vendor-*` groups vite.config.ts splits out) are reported
but not budgeted: they change with dependency bumps rather than with our code,
and they stay browser-cached across releases. The budget is on app code, the
part a lazy boundary moves. It was 273 kB gzip before the drawers, the gate
screens and the Spanish locale moved behind lazy imports (S6-3b); 120 kB is
the plan's ceiling.

Run after `npm run build`. Exit 1 over budget, 2 when there is no build.
"""

import gzip
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
DIST = ROOT / "internal" / "api" / "ui"
BUDGET_BYTES = 120_000

SCRIPT = re.compile(r'(?:src|href)="/assets/([^"]+\.js)"')


def initial_chunks(dist):
    """Every JS file index.html loads, entry first."""
    return SCRIPT.findall((dist / "index.html").read_text(encoding="utf-8"))


def is_vendor(name):
    return name.startswith(("vendor-", "rolldown-runtime-"))


def gzip_size(path):
    return len(gzip.compress(path.read_bytes(), compresslevel=9))


def main(dist=DIST, budget=BUDGET_BYTES, out=sys.stdout):
    if not (dist / "index.html").is_file():
        print(f"no build at {dist}; run `npm run build` in ui/ first", file=out)
        return 2
    app = vendor = 0
    for name in initial_chunks(dist):
        size = gzip_size(dist / "assets" / name)
        if is_vendor(name):
            vendor += size
        else:
            app += size
        print(f"{size / 1000:8.2f} kB  {name}", file=out)
    print(f"first-load app JS {app / 1000:.2f} kB gzip (budget {budget / 1000:.0f} kB); "
          f"vendor {vendor / 1000:.2f} kB", file=out)
    if app > budget:
        print("over budget: move the new first-load code behind a lazy import", file=out)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
