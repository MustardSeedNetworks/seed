#!/usr/bin/env python3
"""Ratchet the UI's hardcoded English copy down, where the shared gate cannot see it.

The shared i18n gate (MustardSeedNetworks/.github, scripts/i18n/check-source.py)
blocks on hardcoded *bare JSX text* — everything between a closing `>` and the
next `<`. That is a real check and it stays. It has one shape-shaped blind spot,
found by seed#2491 and again by seed#2495: copy that never appears as a text
node. All three of these passed it while rendering English to a Spanish operator:

    <IconButton aria-label="Refresh" />              a prop value
    <ListDetail empty="No alerts match this filter" />
    {loading ? 'Loading…' : rows.length}             a string in an expression

So this gate scans the same tree for the two shapes the text-node scan cannot
reach:

  copy-prop   a value of a prop that carries copy (aria-label, placeholder,
              label, empty, title, …). Single words count: `aria-label="Close"`
              is read out by a screen reader either way.
  literal     a prose string literal (two or more words, capitalised) anywhere
              in a .tsx that is not the argument of a t() call. This is the
              shape a `Record<Status, string>` of labels and a thrown
              `new Error('Failed to load users')` both take.

It is a RATCHET against scripts/i18n-copy-baseline.txt, the same shape as the
route-consumer and UI-fetch gates: existing sites are listed with a reason and
may only shrink; a new one fails immediately; an entry that no longer matches
must be removed so the file cannot drift into fiction.

Entries are keyed by file and text, NOT by line number. The UI-fetch baseline is
line-anchored and every split above a baselined site has had to re-point it
(seed#2467 slice 3 broke CI that way twice); copy does not move when the lines
around it do.
"""

from __future__ import annotations

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
BASELINE = ROOT / "scripts" / "i18n-copy-baseline.txt"
SRC = ROOT / "ui" / "src"

SKIP_SUFFIX = (".test.tsx", ".stories.tsx", ".d.ts")

# Props whose value is read by a person or a screen reader. A value here is copy
# however short it is.
COPY_PROPS = (
    "alt", "aria-label", "ariaLabel", "aria-description", "body", "caption",
    "cancelLabel", "confirmLabel", "description", "empty", "emptyMessage",
    "error", "eyebrow", "headline", "heading", "helpText", "hint", "label",
    "leftLabel", "legend", "message", "placeholder", "rightLabel", "subtitle",
    "submitLabel", "summary", "text", "title", "tooltip",
)

PROP = re.compile(
    r"\b(" + "|".join(re.escape(p) for p in COPY_PROPS) + r")=(\"|')([^\"'{}<>\n]{2,120})\2"
)
# A capitalised multi-word string: "Failed to load users". The space is
# required and the delimiters are excluded from the separator class on purpose:
# without them "Content-Type" and the tail of `'Connecting...' : 'Connect'` both
# read as prose, and a gate that cries wolf gets a blanket baseline entry.
LITERAL = re.compile(r"(\"|')([A-Z][a-z]+(?:[ ,.!?:;’/-]+[^\"'\n]{1,80})+?)\1")

# A developer-facing log line, not shipped copy.
LOG_CALL = re.compile(r"\b(?:logger|console)\.\w+\(")

# An example value, not copy: an IP, a port, a host, a URL, a lowercase sample.
EXAMPLE = re.compile(r"^(?:[^A-Z]|.*://|[\w.-]+\.[a-z]{2,}$)")
# `t('key')`, `t("key", …)` — the literal that follows is a key, not copy.
T_CALL = re.compile(r"\bt\(\s*$")


def blank_comments(text: str) -> str:
    """Replace comment bodies with spaces, preserving offsets and line count.

    Prose inside a JSDoc example is not shipped copy; scanning it is what keeps
    a check like this unblockable.
    """
    out = list(text)
    i, n, state, quote = 0, len(text), None, ""
    while i < n:
        c, nxt = text[i], text[i + 1] if i + 1 < n else ""
        if state is None:
            if c in "'\"`":
                state, quote = "str", c
            elif c == "/" and nxt in "/*":
                state = "line" if nxt == "/" else "block"
                out[i] = out[i + 1] = " "
                i += 2
                continue
        elif state == "str":
            if c == "\\":
                i += 2
                continue
            if c == quote:
                state = None
        elif state == "line":
            if c == "\n":
                state = None
            else:
                out[i] = " "
        elif state == "block":
            if c == "*" and nxt == "/":
                out[i] = out[i + 1] = " "
                i += 2
                state = None
                continue
            if c != "\n":
                out[i] = " "
        i += 1
    return "".join(out)


def sites(src: pathlib.Path, root: pathlib.Path):
    """Yield (key, relative path, kind, text) for each hardcoded-copy site."""
    for path in sorted(src.rglob("*.tsx")):
        if path.name.endswith(SKIP_SUFFIX) or "/test/" in str(path):
            continue
        text = blank_comments(path.read_text(encoding="utf-8"))
        rel = path.relative_to(root).as_posix()
        seen = set()
        for match in PROP.finditer(text):
            prop, value = match.group(1), match.group(3).strip()
            if not value or EXAMPLE.match(value):
                continue
            key = f"{rel} copy-prop {prop}={value}"
            if key not in seen:
                seen.add(key)
                yield key, rel, "copy-prop", f"{prop}={value!r}"
        for match in LITERAL.finditer(text):
            value = " ".join(match.group(2).split())
            if " " not in value:
                continue
            # A log line is read by whoever runs the daemon, not by the
            # operator using the UI, and it stays English on purpose.
            line_start = text.rfind("\n", 0, match.start()) + 1
            if LOG_CALL.search(text[line_start : match.start()]):
                continue
            # The first argument of t() is a key, and a key looks nothing like
            # this, but `t('Some key', …)` would still be a false positive.
            if T_CALL.search(text[max(0, match.start() - 40) : match.start()]):
                continue
            key = f"{rel} literal {value}"
            if key not in seen:
                seen.add(key)
                yield key, rel, "literal", repr(value)


def load_baseline(path: pathlib.Path) -> dict[str, str]:
    baseline = {}
    if path.exists():
        for raw in path.read_text(encoding="utf-8").splitlines():
            entry = raw.split("  #", 1)[0].strip()
            if entry and not entry.startswith("#"):
                baseline[entry] = raw
    return baseline


def main() -> int:
    found = {key: (rel, kind, shown) for key, rel, kind, shown in sites(SRC, ROOT)}
    baseline = load_baseline(BASELINE)

    new = sorted(set(found) - set(baseline))
    gone = sorted(set(baseline) - set(found))

    failed = False
    if new:
        failed = True
        print("::error::hardcoded English copy the shared JSX-text gate cannot see "
              "— move it into a locale file and read it with t():")
        for key in new:
            rel, kind, shown = found[key]
            print(f"  {kind:9} {rel}: {shown}")
            print(f"::error file={rel}::hardcoded English copy ({kind}): {shown}")
    if gone:
        failed = True
        print(f"::error::baseline entries that no longer match — remove them from {BASELINE}:")
        for key in gone:
            print(f"  {key}")

    props = sum(1 for _, kind, _ in found.values() if kind == "copy-prop")
    print(
        f"i18n copy gate: {len(found)} hardcoded sites "
        f"({props} copy-prop, {len(found) - props} literal), {len(baseline)} baselined."
    )
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
