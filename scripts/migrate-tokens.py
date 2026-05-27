#!/usr/bin/env python3
"""
migrate-tokens.py — bulk Tailwind→semantic-token migration.

Replaces raw Tailwind utility classes with semantic tokens defined in
ui/src/index.css `@layer components` and ui/src/styles/. Idempotent.

Scope: every *.tsx and *.ts file under ui/src/ EXCEPT styles/, constants/,
and test files (*.test.tsx, *.spec.tsx, *.stories.tsx, *.mock.tsx).

Run from repo root: scripts/migrate-tokens.py
"""

from __future__ import annotations

import re
import sys
from collections import Counter
from pathlib import Path

# Single-class swaps — keys are old Tailwind utilities, values are semantic tokens.
# Each swap is applied with a lookbehind/lookahead that rejects neighboring
# word/hyphen chars, so `space-y-3` matches but `aria-space-y-3` does not.
SINGLE_CLASS_SWAPS: dict[str, str] = {
    # space-y → stack
    "space-y-1": "stack-xs",
    "space-y-2": "stack-sm",
    "space-y-3": "stack",
    "space-y-4": "stack-lg",
    "space-y-6": "stack-xl",
    # gap-N → gap-*
    "gap-1": "gap-tight",
    "gap-2": "gap-compact",
    "gap-3": "gap-default",
    "gap-4": "gap-comfortable",
    "gap-6": "gap-spacious",
    # p-N (standalone, not px/py/pt/pr/pb/pl) → pad-*
    "p-2": "pad-xs",
    "p-3": "pad-sm",
    "p-4": "pad",
    "p-6": "pad-lg",
    "p-8": "pad-xl",
    # mb-N → mb-*
    "mb-1": "mb-tight",
    "mb-3": "mb-heading",
    "mb-4": "mb-content",
    "mb-6": "mb-section",
    "mb-8": "mb-section-lg",
    # mt-N → mt-*
    "mt-1": "mt-tight",
    "mt-2": "mt-inline",
    "mt-3": "mt-heading",
    "mt-4": "mt-content",
    "mt-8": "mt-section",
    # ml-N → ml-*
    "ml-1": "ml-tight",
    "ml-2": "ml-inline",
    "ml-4": "ml-content",
    "ml-6": "ml-spacious",
    # pt-N → pt-*
    "pt-1": "pt-tight",
    "pt-3": "pt-heading",
    "pt-4": "pt-section",
    # pb-N → pb-*
    "pb-1": "pb-tight",
    "pb-2": "pb-inline",
    # px-2 → px-cell
    "px-2": "px-cell",
    # pr / pl / py specials
    "pr-10": "pr-icon",
    "pr-8": "pr-tight",
    "pl-5": "pl-indent",
    "py-12": "py-centered",
    "py-1": "py-compact",
    "py-1.5": "py-compact-md",
    "py-2": "py-row",
    "py-3": "py-row-lg",
    # font weights already exist but are uniform Tailwind — leave as is
    # (they're tokenized via theme.typography.weight.* and theme is OK).
}

# Tailwind v4 arbitrary-value patterns that bypass the theme indirection.
# `text-[var(--color-status-error)]` → `text-status-error` (and same for bg/border/ring).
ARBITRARY_VAR_PATTERN = re.compile(
    r"\b(text|bg|border|ring|fill|stroke|placeholder|shadow|from|via|to|outline|divide|accent|caret|decoration)-\[var\(--color-([a-z][a-z0-9-]*)\)\]"
)

# Compound patterns — multi-class strings that map to a single semantic token.
# Order-sensitive: matches must appear as written (post Biome formatting they
# normally do). Anything that survives can be hand-fixed.
COMPOUND_PATTERNS: list[tuple[str, str]] = [
    # Headings — full composition (semantic class re-applies the rest).
    (r"text-2xl font-bold text-text-primary leading-tight tracking-tight", "heading-1"),
    (r"text-xl font-semibold text-text-primary leading-snug", "heading-2"),
    (r"text-lg font-semibold text-text-primary leading-snug", "heading-3"),
    (r"text-base font-medium text-text-primary leading-snug", "heading-4"),
    # Section / category label
    (
        r"text-xs font-medium uppercase tracking-wider text-text-muted",
        "section-title",
    ),
    # Body
    (r"text-lg text-text-primary leading-relaxed", "body-large"),
    (r"text-base text-text-primary leading-relaxed", "body"),
    (r"text-sm text-text-secondary leading-relaxed", "body-small"),
    (r"text-xs text-text-muted leading-normal", "caption"),
    # Label
    (r"text-sm font-medium text-text-primary", "label"),
    # Flex shortcuts
    (r"flex items-center justify-between", "flex-between"),
    (r"flex items-center justify-center", "flex-center"),
]

# Forgiving heading pair-matchers (size + weight only, in either order).
# Semantic class adds the missing leading/tracking/color — small visual change
# is acceptable per the harmonization principle (consistent typography wins).
HEADING_PAIRS: list[tuple[str, str]] = [
    (r"text-2xl font-bold", "heading-1"),
    (r"font-bold text-2xl", "heading-1"),
    (r"text-xl font-semibold", "heading-2"),
    (r"font-semibold text-xl", "heading-2"),
    (r"text-lg font-semibold", "heading-3"),
    (r"font-semibold text-lg", "heading-3"),
    (r"text-base font-medium", "heading-4"),
    (r"font-medium text-base", "heading-4"),
]


def neighbor_safe(old: str) -> re.Pattern[str]:
    """Build a regex that matches `old` only when surrounded by non-word, non-hyphen, non-dot chars.

    Excluding `.` is critical: without it, `py-1` would match the prefix of
    `py-1.5` (since `.` is neither `\\w` nor `-`), producing `py-compact.5`.
    """
    return re.compile(rf"(?<![-\w.]){re.escape(old)}(?![-\w.])")


def compound_safe(pattern: str) -> re.Pattern[str]:
    """Compound patterns. Match the literal sequence with non-word, non-hyphen, non-dot boundaries."""
    return re.compile(rf"(?<![-\w.]){re.escape(pattern)}(?![-\w.])")


def gather_targets(root: Path) -> list[Path]:
    """Return every *.tsx / *.ts file under root/ui/src minus excluded paths."""
    ui_src = root / "ui" / "src"
    if not ui_src.is_dir():
        raise SystemExit(f"ERROR: {ui_src} not found. Run from repo root.")
    files: list[Path] = []
    for p in ui_src.rglob("*"):
        if not p.is_file():
            continue
        if p.suffix not in {".ts", ".tsx"}:
            continue
        s = str(p)
        if any(seg in s for seg in ("/styles/", "/constants/")):
            continue
        if any(
            s.endswith(suffix)
            for suffix in (".test.ts", ".test.tsx", ".spec.ts", ".spec.tsx", ".stories.tsx", ".mock.ts", ".mock.tsx")
        ):
            continue
        files.append(p)
    return files


def main() -> int:
    root = Path.cwd()
    targets = gather_targets(root)
    print(f"Scanning {len(targets)} files under {root / 'ui' / 'src'}")

    # Pre-compile patterns. Apply compound BEFORE single — compound contains
    # substrings that would otherwise be partially rewritten by single-class
    # passes (e.g. text-2xl alone shouldn't change, but the compound heading
    # pattern containing text-2xl should).
    compound_compiled = [(compound_safe(pat), repl) for pat, repl in COMPOUND_PATTERNS]
    heading_compiled = [(compound_safe(pat), repl) for pat, repl in HEADING_PAIRS]
    # Sort single-class swaps by source length DESC so longer keys (py-1.5)
    # match before shorter prefixes (py-1). Combined with the non-dot
    # lookahead in neighbor_safe, this prevents prefix-collision bugs.
    sorted_singles = sorted(SINGLE_CLASS_SWAPS.items(), key=lambda kv: -len(kv[0]))
    single_compiled = [(neighbor_safe(old), repl) for old, repl in sorted_singles]

    total_changes: Counter[str] = Counter()
    files_changed = 0

    for file in targets:
        original = file.read_text(encoding="utf-8")
        text = original

        # 1. Tailwind v4 arbitrary-value indirection: text-[var(--color-X)] → text-X.
        def _arb_repl(m: re.Match[str]) -> str:
            total_changes[f"{m.group(1)}-{m.group(2)} (from var)"] += 1
            return f"{m.group(1)}-{m.group(2)}"

        text = ARBITRARY_VAR_PATTERN.sub(_arb_repl, text)

        # 2. Compound multi-class patterns (must run before pair / single swaps).
        for pat, repl in compound_compiled:
            new_text, n = pat.subn(repl, text)
            if n:
                total_changes[repl] += n
                text = new_text

        # 3. Heading pairs (size + weight, either order).
        for pat, repl in heading_compiled:
            new_text, n = pat.subn(repl, text)
            if n:
                total_changes[repl] += n
                text = new_text

        # 4. Single-class swaps.
        for pat, repl in single_compiled:
            new_text, n = pat.subn(repl, text)
            if n:
                total_changes[repl] += n
                text = new_text

        if text != original:
            file.write_text(text, encoding="utf-8")
            files_changed += 1

    print(f"\nFiles changed: {files_changed}")
    print(f"Total replacements: {sum(total_changes.values())}")
    if total_changes:
        print("\nBy token (top 20):")
        for tok, n in total_changes.most_common(20):
            print(f"  {n:>5}  {tok}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
