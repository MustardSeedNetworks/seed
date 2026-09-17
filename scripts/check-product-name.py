#!/usr/bin/env python3
"""The product is Seed, not "The Seed", on every surface a customer reads.

Owner decision 2026-09-15 (fleet): drop the article in-app, in all four
products. UI-SEED-9 renamed 104 occurrences here; without a gate they come
back one string at a time, which is how 72 of them accumulated in the first
place.

Two shapes are rejected, both case-SENSITIVE:

    The Seed     the article plus the name, in prose or a label
    THE SEED     the same in an upper-case banner (three CLI boxes had one)

Case-sensitivity is the whole design. The obvious home for this rule was
`scripts/i18n/banned-vocab.txt`, and stem tried exactly that in UI-STEM-8 and
reverted it: that matcher folds case, so `the seed` matches ordinary prose
("restart the seed daemon") and the check drowned in findings. Matching only a
capital `The` keeps the prose and catches the brand. A sentence that genuinely
begins "The Seed daemon ..." trips this; rewrite it rather than exempt it —
sentence-initial is precisely where the article crept back in.

Scope is the surfaces the rename covered: the UI, the locales, the CLI, the
installers and the config schema. Deliberately NOT scanned:

  internal/auth       `Issuer` is the JWT `iss` claim; renaming it invalidates
                      every live token. Same call stem made for `defaultIssuer`.
  User-Agent strings  wire identifiers a peer or a log parser may match on.
  TLS cert subject    `Organization`/`CommonName` are pinnable identifiers.
  docs/, README, CHANGELOG, LICENSE
                      not in-app; the decision says "everywhere in-app", and
                      CHANGELOG is history that must not be rewritten.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

DEFAULT_ROOT = Path(__file__).resolve().parent.parent

# Files and directories that ship user-visible product-name copy.
SCOPE: tuple[str, ...] = (
    "ui/src",
    "ui/e2e",
    "ui/index.html",
    "internal/i18n",
    "cmd/seed",
    "deploy",
    "internal/config/schema.json",
    "package.json",
)

SKIP_DIRS = {"node_modules", "dist", "coverage", ".git"}
BINARY_SUFFIXES = {".png", ".jpg", ".jpeg", ".gif", ".ico", ".icns", ".woff", ".woff2", ".pdf"}

PATTERN = re.compile(r"The Seed|THE SEED")


def candidates(root: Path) -> list[Path]:
    files: list[Path] = []
    for entry in SCOPE:
        path = root / entry
        if path.is_file():
            files.append(path)
        elif path.is_dir():
            files += [
                f
                for f in path.rglob("*")
                if f.is_file()
                and f.suffix.lower() not in BINARY_SUFFIXES
                and not SKIP_DIRS & set(f.relative_to(root).parts)
            ]
    return sorted(files)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=DEFAULT_ROOT, help="repository root to scan")
    args = parser.parse_args()

    files = candidates(args.root)
    findings: list[str] = []
    for file in files:
        try:
            text = file.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue
        for number, line in enumerate(text.splitlines(), 1):
            if PATTERN.search(line):
                findings.append(f"  {file.relative_to(args.root)}:{number}: {line.strip()[:100]}")

    if findings:
        print('FAIL: the product is "Seed"; the article is back on a customer-facing surface:')
        print("\n".join(findings))
        print('\nDrop the article (owner decision 2026-09-15, fleet: Seed, Stem, NIAC, Trellis).')
        return 1

    print(f"OK: no articled product name on {len(files)} in-app files.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
