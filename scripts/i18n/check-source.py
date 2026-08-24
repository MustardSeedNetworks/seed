#!/usr/bin/env python3
"""Source-side i18n checks that a line-oriented grep cannot do correctly.

This check used to be a `grep -rnE` one-liner in validate.sh. TypeScript
and JSX are not line-oriented, and the greps missed accordingly:

  * the fallback check could not see the multiline form, so ten
    `t(\\n 'key',\\n 'fallback',\\n)` sites sat in the tree unreported, and it
    mis-parsed a fallback whose delimiter it did not own
    (`"Choose this stem's role"` — an apostrophe inside double quotes);

  * the hardcoded-English check only matched text beginning on the same line
    as the closing `>`, so it reported one of AuthGate's four English strings
    and stayed quiet about "Sign in to continue".

Comments are blanked before scanning — prose inside a JSDoc example is not
shipped copy, and treating it as a finding is what kept the hardcoded-English
check unblockable.

Exit status is 1 if anything is found, 0 otherwise. Findings are printed as
GitHub Actions annotations.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
UI_SRC = ROOT / "ui" / "src"

SKIP_PARTS = ("/node_modules/", "/test/", "/__stories__/")
SKIP_SUFFIX = (".d.ts", ".test.ts", ".test.tsx", ".stories.tsx")

# A JSX text node: everything between a closing '>' and the next '<' that
# contains no braces (an interpolation means the text is already dynamic) and
# no angle brackets (that would be another tag).
TEXT_NODE = re.compile(r">([^<>{}]+)<", re.DOTALL)

# English prose: two or more words, the first being Capitalised-then-lowercase.
# "RFC 2544 Switch Tests" is not matched (RF is two capitals), nor is a bare
# "Reflector", nor lowercase code fragments.
PROSE = re.compile(r"^[A-Z][a-z]+(?:[\s,.!?:;'’-]+\S+)+", re.DOTALL)


def blank_comments(text: str) -> str:
    """Replace comment bodies with spaces, preserving offsets and line count."""
    out = list(text)
    i, n = 0, len(text)
    state = None  # None | 'line' | 'block' | 'str'
    quote = ""
    while i < n:
        c = text[i]
        nxt = text[i + 1] if i + 1 < n else ""
        if state is None:
            if c in "'\"`":
                state, quote = "str", c
            elif c == "/" and nxt == "/":
                state = "line"
                out[i] = out[i + 1] = " "
                i += 2
                continue
            elif c == "/" and nxt == "*":
                state = "block"
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


def sources() -> list[Path]:
    files = []
    for p in sorted(UI_SRC.rglob("*.ts*")):
        s = str(p)
        if any(part in s for part in SKIP_PARTS) or s.endswith(SKIP_SUFFIX):
            continue
        files.append(p)
    return files


def main() -> int:
    hardcoded: list[tuple[str, int, str]] = []

    for path in sources():
        try:
            raw = path.read_text()
        except UnicodeDecodeError:
            continue
        text = blank_comments(raw)
        rel = path.relative_to(ROOT)

        if path.suffix == ".tsx":
            for m in TEXT_NODE.finditer(text):
                inner = m.group(1).strip()
                if not inner or not PROSE.match(inner):
                    continue
                line = raw[: m.start(1)].count("\n") + 1
                # The marker may sit on the text's own line or on a comment
                # line just above it, which is where it reads naturally.
                window = raw.split("\n")[max(0, line - 4) : line]
                if any("allow-hardcoded" in w for w in window):
                    continue
                hardcoded.append((str(rel), line, " ".join(inner.split())[:100]))

    for label, found, hint in (
        ("hardcoded English string", hardcoded, "move the copy into a locale file"),
    ):
        if found:
            print(f"::error::{len(found)} {label}(s) — {hint}:")
            for f, line, snippet in found:
                print(f"  {f}:{line}: {snippet}")
                print(f"::error file={f},line={line}::{label} {hint}")

    return 1 if hardcoded else 0


if __name__ == "__main__":
    sys.exit(main())
