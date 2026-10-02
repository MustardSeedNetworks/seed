#!/usr/bin/env python3
"""Fail on a Tailwind variant applied to a class defined in @layer components.

Tailwind v4 generates variants only for utilities. A class defined inside
`@layer components` is plain CSS, so `sm:pad-lg` or `focus:py-row` compiles to
nothing: the markup claims a responsive or state style the bundle never ships
(.github#80, where the shell padding sat at 16px under a `sm:pad-lg
lg:pad-xl` that had never applied). Either drop the variant or use a raw
utility, which Tailwind does generate variants for.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

COMMENT = re.compile(r"/\*.*?\*/", re.S)
LAYER = re.compile(r"@layer\s+components\s*\{")
CLASS = re.compile(r"\.(-?[A-Za-z_][\w-]*)")
TOKEN_SPLIT = re.compile(r"[\s\"'`{}(),]+")


def component_classes(css: str) -> set[str]:
    """Class names in the selectors of every rule inside @layer components."""
    css = COMMENT.sub("", css)
    names: set[str] = set()
    for layer in LAYER.finditer(css):
        depth = 1
        prelude_start = layer.end()
        for i in range(layer.end(), len(css)):
            char = css[i]
            if char == "{":
                prelude = css[prelude_start:i].strip()
                if not prelude.startswith("@"):
                    names.update(CLASS.findall(prelude))
                depth += 1
            elif char == "}":
                depth -= 1
                if depth == 0:
                    break
            if char in "{};":
                prelude_start = i + 1
    return names


def variant_hits(source: str, names: set[str]) -> list[tuple[int, str]]:
    hits = []
    for number, line in enumerate(source.splitlines(), 1):
        for token in TOKEN_SPLIT.split(line):
            *variants, base = token.split(":")
            if variants and all(variants) and base.lstrip("!") in names:
                hits.append((number, token))
    return hits


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parent.parent)
    src = parser.parse_args().root / "ui" / "src"

    names: set[str] = set()
    for css in sorted(src.rglob("*.css")):
        names |= component_classes(css.read_text(encoding="utf-8"))

    failures = []
    for path in sorted(p for p in src.rglob("*") if p.suffix in {".ts", ".tsx"}):
        rel = path.relative_to(src.parent.parent)
        for number, token in variant_hits(path.read_text(encoding="utf-8"), names):
            failures.append(f"{rel}:{number}: {token}")

    if not failures:
        print(f"No variants on the {len(names)} @layer components classes.")
        return 0
    print("::error::Variants on @layer components classes compile to nothing:")
    print("\n".join(failures))
    print("\nDrop the variant, or use the raw utility it stands for (focus:py-2).")
    return 1


if __name__ == "__main__":
    sys.exit(main())
