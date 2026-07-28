#!/usr/bin/env python3
"""Reject locked customer-facing vocabulary in the tracked repository tree."""

from __future__ import annotations

import argparse
import fnmatch
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Config:
    root: Path
    terms: Path
    exclusions: Path


def parse_args() -> Config:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path)
    parser.add_argument("--terms", type=Path)
    parser.add_argument("--exclusions", type=Path)
    args = parser.parse_args()
    root = args.root or Path(__file__).resolve().parent.parent
    terms = args.terms or root / "scripts/i18n/banned-vocab.txt"
    exclusions = args.exclusions or root / "scripts/banned-vocabulary-exclusions.txt"
    return Config(root.resolve(), terms.resolve(), exclusions.resolve())


def meaningful_lines(path: Path) -> list[str]:
    if not path.is_file():
        raise ValueError(f"required policy file is missing: {path}")
    lines = path.read_text(encoding="utf-8").splitlines()
    return [
        line.strip()
        for line in lines
        if line.strip() and not line.lstrip().startswith("#")
    ]


def load_terms(path: Path) -> list[str]:
    terms = meaningful_lines(path)
    if not terms:
        raise ValueError(f"term policy is empty: {path}")
    return terms


def load_exclusions(path: Path) -> list[str]:
    entries = meaningful_lines(path)
    patterns: list[str] = []
    for entry in entries:
        parts = [part.strip() for part in entry.split("|", 1)]
        if len(parts) != 2 or not all(parts):
            raise ValueError(f"exclusion needs 'pattern | reason': {entry}")
        patterns.append(parts[0])
    return patterns


def tracked_files(root: Path) -> list[str]:
    result = subprocess.run(
        ["git", "-C", str(root), "ls-files", "-z"],
        check=True,
        capture_output=True,
    )
    return sorted(item.decode() for item in result.stdout.split(b"\0") if item)


def excluded(path: str, patterns: list[str]) -> bool:
    return any(fnmatch.fnmatchcase(path, pattern) for pattern in patterns)


def patterns_for(terms: list[str]) -> list[tuple[str, re.Pattern[str]]]:
    return [
        (
            term,
            re.compile(rf"(?<![A-Za-z]){re.escape(term)}(?![A-Za-z])", re.IGNORECASE),
        )
        for term in terms
    ]


def text_lines(path: Path) -> list[str]:
    if path.is_symlink() or not path.is_file():
        return []
    content = path.read_bytes()
    if b"\0" in content:
        return []
    return content.decode("utf-8", errors="replace").splitlines()


def scan(config: Config) -> list[tuple[str, int, str]]:
    terms = patterns_for(load_terms(config.terms))
    exclusions = load_exclusions(config.exclusions)
    findings: list[tuple[str, int, str]] = []
    for relative in tracked_files(config.root):
        if excluded(relative, exclusions):
            continue
        for number, line in enumerate(text_lines(config.root / relative), start=1):
            findings.extend(
                (relative, number, term)
                for term, pattern in terms
                if pattern.search(line)
            )
    return sorted(findings)


def main() -> int:
    try:
        findings = scan(parse_args())
    except (OSError, subprocess.CalledProcessError, UnicodeError, ValueError) as error:
        print(f"banned-vocabulary checker error: {error}", file=sys.stderr)
        return 2
    for path, line, term in findings:
        print(f"{path}:{line}: banned term '{term}'")
    if findings:
        print(f"banned-vocabulary check failed: {len(findings)} finding(s)")
        return 1
    print("banned-vocabulary check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
