#!/usr/bin/env python3
"""check-route-consumers.py — route ⇄ consumer plumbing gate.

The route registry says what the API serves; the UI and CLI clients say what
is asked for. Nothing checked that the two agree, so the 2026-09-02 audit
found routes with no caller and UI calls with no route in both products.

Two checks:

  A. Every API path the UI (ui/src, excluding tests and stories) or a Go
     client (cmd/, internal/cliclient) requests matches a registered route.
     A miss is a live 404. Known ones are baselined as `404 <path>  # issue`
     so the gate can land green while the fix is tracked; a new one fails.
  B. Every registered route has at least one consumer. Routes without one are
     a RATCHET against scripts/route-consumer-baseline.txt: growth fails, and a
     baselined route that has since gained a consumer fails too, so the
     baseline stays a work queue rather than an allow-list.

This does not prove a handler dispatches correctly inside a prefix route —
seed's profile router bug hid behind a matching `/profiles/` prefix. Only a
test through the real mux catches that class.

Run locally: scripts/check-route-consumers.py   ·   regenerate: --update
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

API_PREFIX = "/api/v1"
BASELINE = "scripts/route-consumer-baseline.txt"

ROUTE_LITERAL = re.compile(r'path:\s*(APIVersionPrefix\s*\+\s*)?"([^"]+)"')
DIRECT_HANDLE = re.compile(r'HandleFunc\("(/api/[^"]+)"')
SESSION_RESOURCE = re.compile(r'^\s*"([\w-]+)":\s*s\.handle', re.M)
# Consumers are scanned for any `/api/` path, not only `/api/v1/`: a call to a
# pre-versioning path is exactly the kind of miss this gate exists to find.
# The path must start a string (or follow a `${base}` interpolation) so that
# import specifiers such as '../api/client' are not read as requests.
PATH_LITERAL = re.compile(r"""(?<=['"`}])/api/[a-z0-9][^'"`\s)]*""")
TS_CONST = re.compile(r"""const\s+(\w+)\s*=\s*['"](/api/[^'"]*)['"]""")
TS_CONST_USE = re.compile(r"`\$\{(\w+)\}([^`]*)`")
COMMENT_LINE = re.compile(r"^\s*(//|/\*|\*)")
TS_EXCLUDE = re.compile(r"\.(test|spec|stories)\.tsx?$|/__mocks__/|/test/|/e2e/|ui/src/data/|ui/src/i18n/")


def normalise(path: str) -> str:
    """Reduce a consumer path to a matchable pattern: drop the query string and
    turn every interpolated or formatted segment into a wildcard."""
    path = path.split("?", 1)[0]
    path = re.sub(r"\$\{[^}]*\}", "*", path)
    path = re.sub(r"%[sdv]", "*", path)
    path = re.sub(r"/\*[^/]*", "/*", path)
    path = re.sub(r"(?<=[^/])\*$", "", path)  # `${ENDPOINT}${query}` — a glued trailing wildcard is a query builder
    return path


def load_routes(root: Path) -> set[str]:
    routes: set[str] = set()
    for file in sorted((root / "internal" / "api").glob("*.go")):
        if file.name.endswith("_test.go"):
            continue
        text = file.read_text(encoding="utf-8")
        for prefixed, path in ROUTE_LITERAL.findall(text):
            if " " in path:  # "GET /api/v1/x" style
                path = path.split(" ", 1)[1]
            if prefixed:
                path = API_PREFIX + path
            if path.startswith("/api/"):
                routes.add(path)
        routes.update(DIRECT_HANDLE.findall(text))
        if file.name == "handlers_sessions.go":
            for resource in SESSION_RESOURCE.findall(text):
                routes.add(f"{API_PREFIX}/sessions/*/{resource}")
    return routes


def _ts_files(root: Path):
    ui = root / "ui" / "src"
    for file in sorted(ui.rglob("*")):
        if file.suffix in {".ts", ".tsx"} and not TS_EXCLUDE.search(file.as_posix()):
            yield file


def _go_client_files(root: Path):
    for base in (root / "cmd", root / "internal" / "cliclient"):
        if base.is_dir():
            for file in sorted(base.rglob("*.go")):
                if not file.name.endswith("_test.go"):
                    yield file


def load_consumers(root: Path) -> dict[str, set[str]]:
    consumers: dict[str, set[str]] = {}

    def add(path: str, where: str) -> None:
        consumers.setdefault(normalise(path), set()).add(where)

    ts_texts = {file: file.read_text(encoding="utf-8") for file in _ts_files(root)}
    # Endpoint constants (`const ENDPOINT = '/api/v1/x'`) are resolved per file
    # first — several hooks reuse the name ENDPOINT for different paths — and
    # repo-wide as a fallback for the few that are exported and imported.
    shared: dict[str, str] = {}
    for text in ts_texts.values():
        shared.update(TS_CONST.findall(text))
    for file, text in ts_texts.items():
        rel = file.relative_to(root).as_posix()
        consts = dict(shared, **dict(TS_CONST.findall(text)))
        for lineno, line in enumerate(text.splitlines(), 1):
            if COMMENT_LINE.match(line) or TS_CONST.search(line):
                continue
            for path in PATH_LITERAL.findall(line):
                add(path, f"{rel}:{lineno}")
            for name, suffix in TS_CONST_USE.findall(line):
                if name in consts:
                    add(consts[name] + suffix, f"{rel}:{lineno}")
            for name, value in consts.items():
                # `api.get(ENDPOINT)` — the constant used bare, not as `${ENDPOINT}/more`.
                if re.search(rf"(?<!\$\{{)\b{re.escape(name)}\b(?![\w}}])", line):
                    add(value, f"{rel}:{lineno}")
    for file in _go_client_files(root):
        rel = file.relative_to(root).as_posix()
        for lineno, line in enumerate(file.read_text(encoding="utf-8").splitlines(), 1):
            if COMMENT_LINE.match(line):
                continue
            for path in PATH_LITERAL.findall(line):
                add(path, f"{rel}:{lineno}")
    return consumers


def is_base_of(route: str, consumer: str) -> bool:
    """A consumer that stops at a strict segment-prefix of a route is a base URL
    that gets a suffix elsewhere (`${base}/packets`), not a request."""
    rs = [s for s in route.split("/") if s]
    cs = [s for s in consumer.split("/") if s]
    return not consumer.endswith("/") and len(cs) < len(rs) and all(a == b or "*" in (a, b) for a, b in zip(rs, cs))


def matches(route: str, consumer: str) -> bool:
    route_prefix = route.endswith("/")
    consumer_prefix = consumer.endswith("/")
    rs = [s for s in route.split("/") if s]
    cs = [s for s in consumer.split("/") if s]
    if route_prefix and consumer_prefix:
        n = min(len(rs), len(cs))
        return all(a == b or "*" in (a, b) for a, b in zip(rs[:n], cs[:n]))
    if route_prefix:
        return len(cs) >= len(rs) and all(a == b or "*" in (a, b) for a, b in zip(rs, cs))
    if consumer_prefix:
        return len(rs) >= len(cs) and all(a == b or "*" in (a, b) for a, b in zip(rs, cs))
    return len(rs) == len(cs) and all(a == b or "*" in (a, b) for a, b in zip(rs, cs))


def read_baseline(path: Path) -> tuple[dict[str, str], dict[str, str]]:
    """Returns (orphaned routes, known 404 request paths), each path -> reason."""
    orphans: dict[str, str] = {}
    known_404: dict[str, str] = {}
    if not path.exists():
        return orphans, known_404
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        key, _, reason = line.partition("#")
        key = key.strip()
        if key.startswith("404 "):
            known_404[key[4:].strip()] = reason.strip()
        else:
            orphans[key] = reason.strip()
    return orphans, known_404


def write_baseline(path: Path, orphans: list[str], no_route: list[str],
                   previous: dict[str, str], previous_404: dict[str, str]) -> None:
    lines = ["# Route ⇄ consumer ratchet; see check-route-consumers.py.",
             "# Plain lines: registered routes with no UI or CLI consumer.",
             "# `404 <path>` lines: requests the UI or CLI makes that no route serves.",
             "# Optional `# reason`. Remove a line when it stops being true — the gate fails on stale entries."]
    for route in sorted(orphans):
        reason = previous.get(route, "")
        lines.append(f"{route}  # {reason}" if reason else route)
    for req in sorted(no_route):
        reason = previous_404.get(req, "")
        lines.append(f"404 {req}  # {reason}" if reason else f"404 {req}")
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def run(root: Path, baseline_path: Path, update: bool = False, out=sys.stdout) -> int:
    routes = load_routes(root)
    consumers = load_consumers(root)
    if not routes:
        print("::error::no routes found under internal/api — is the registry regex stale?", file=out)
        return 2

    no_route = {c: w for c, w in consumers.items()
                if not any(matches(r, c) or is_base_of(r, c) for r in routes)}
    orphans = sorted(r for r in routes if not any(matches(r, c) for c in consumers))
    previous, previous_404 = read_baseline(baseline_path)

    if update:
        write_baseline(baseline_path, orphans, sorted(no_route), previous, previous_404)
        print(f"Wrote {len(orphans)} orphaned route(s) and {len(no_route)} known 404(s) to {baseline_path}", file=out)
        return 0

    failed = False
    new_404 = {c: w for c, w in no_route.items() if c not in previous_404}
    if new_404:
        failed = True
        print("::error::UI/CLI requests with no registered route (live 404s):", file=out)
        for path, where in sorted(new_404.items()):
            print(f"  {path}  <- {', '.join(sorted(where))}", file=out)
    new = [r for r in orphans if r not in previous]
    stale = [r for r in previous if r not in orphans] + [f"404 {c}" for c in previous_404 if c not in no_route]
    if new:
        failed = True
        print("::error::registered routes with no consumer that are not in the baseline "
              "(wire a caller, delete the route, or add to the baseline with a reason):", file=out)
        for r in new:
            print(f"  {r}", file=out)
    if stale:
        failed = True
        print(f"::error::baseline entries that now have a consumer or no longer exist — remove them from {baseline_path}:", file=out)
        for r in stale:
            print(f"  {r}", file=out)
    print(f"Route-consumer gate: {len(routes)} routes, {len(consumers)} consumer paths, "
          f"{len(orphans)} orphaned ({len(previous)} baselined), "
          f"{len(no_route)} without a route ({len(previous_404)} baselined).", file=out)
    return 1 if failed else 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--update", action="store_true", help="rewrite the baseline from the current tree")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    args = parser.parse_args()
    return run(args.root, args.root / BASELINE, update=args.update)


if __name__ == "__main__":
    sys.exit(main())
