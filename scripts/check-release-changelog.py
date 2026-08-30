#!/usr/bin/env python3
"""Verify a release's changelog section against the commits the tag contains.

release-please regenerates the pending release PR's changelog when main moves.
If a commit lands *after* that regeneration and the release PR merges before
the next one, the tag contains the commit and the changelog does not.

That happened on 2026-08-29: seed#2211 shipped in v0.213.38 and appears in no
changelog section at all. Nothing in the release notes records that the
nightly SNMP fail-closed guard shipped, and a bisect driven by changelogs
points at the wrong release.

This compares the two sources directly -- `git log <prev>..<tag>` against the
CHANGELOG section for <tag> -- so the record cannot drift from git silently
regardless of merge ordering.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent

# `type(scope)!: subject (#123)` -- scope, breaking marker and PR are optional.
CONVENTIONAL = re.compile(r"^(?P<type>[a-z]+)(?:\([^)]*\))?!?:\s*(?P<subject>.+)$")
PR_REF = re.compile(r"\(#(?P<pr>\d+)\)\s*$")
SEMVER_TAG = re.compile(r"^v\d+\.\d+\.\d+$")

# release-please writes its own merge commit as `chore(main): release X.Y.Z`.
# It is the release, not a change in it.
RELEASE_COMMIT = re.compile(r"^chore\(main\):\s*release\b")


class NoSection(Exception):
    """The changelog has no section for this tag at all."""


def run(*args: str) -> str:
    return subprocess.run(
        args, cwd=REPO_ROOT, check=True, capture_output=True, text=True
    ).stdout.strip()


def section_titles() -> dict[str, str]:
    """Commit type -> the changelog heading release-please files it under."""
    config = json.loads(
        (REPO_ROOT / ".github" / "release-please-config.json").read_text()
    )
    return {
        s["type"]: s["section"]
        for s in config.get("changelog-sections", [])
        if not s.get("hidden", False)
    }


def changelog_types() -> set[str]:
    """The commit types release-please puts in the changelog.

    Read from the config rather than hardcoded: a type added there must start
    being checked here, without anyone remembering to edit two files.
    """
    config = json.loads(
        (REPO_ROOT / ".github" / "release-please-config.json").read_text()
    )
    return {
        section["type"]
        for section in config.get("changelog-sections", [])
        if not section.get("hidden", False)
    }


def sorted_tags() -> list[str]:
    tags = [t for t in run("git", "tag", "--list").splitlines() if SEMVER_TAG.match(t)]
    return sorted(tags, key=lambda t: tuple(int(p) for p in t[1:].split(".")))


def previous_tag(tag: str) -> str | None:
    tags = sorted_tags()
    if tag not in tags:
        sys.exit(f"{tag} is not a semver tag in this repository")
    index = tags.index(tag)
    return tags[index - 1] if index > 0 else None


def changelog_section(version: str) -> str | None:
    """The CHANGELOG body for one version, or None if it has no section."""
    text = (REPO_ROOT / "CHANGELOG.md").read_text()
    # Headings are `## [0.213.38](...compare...) (date)`.
    start = re.search(rf"^## \[{re.escape(version)}\]", text, re.MULTILINE)
    if start is None:
        return None
    rest = text[start.end():]
    nxt = re.search(r"^## \[", rest, re.MULTILINE)
    return rest[: nxt.start()] if nxt else rest


def repo_slug() -> str:
    url = run("git", "remote", "get-url", "origin")
    return re.sub(r"^.*[:/]([^/]+/[^/]+?)(?:\.git)?$", r"\1", url)


def render_entry(sha: str, subject: str, slug: str) -> str:
    """One changelog line in release-please's own format."""
    body = subject
    scope = re.match(r"^[a-z]+\(([^)]*)\)!?:\s*(.+)$", subject)
    plain = re.match(r"^[a-z]+!?:\s*(.+)$", subject)
    if scope:
        body = f"**{scope.group(1)}:** {scope.group(2)}"
    elif plain:
        body = plain.group(1)
    pr = PR_REF.search(body)
    if pr:
        n = pr.group("pr")
        body = PR_REF.sub(
            f"([#{n}](https://github.com/{slug}/issues/{n}))", body
        )
    full = run("git", "rev-parse", sha)
    return f"* {body} ([{sha}](https://github.com/{slug}/commit/{full}))"


def apply_fix(tag: str, missing: list[tuple[str, str]]) -> None:
    """Append the missing commits to this version's changelog section.

    Entries go under a heading that names why they are here. Silently blending
    them into the generated sections would hide that the record was corrected,
    and the correction is the interesting part.
    """
    version = tag.lstrip("v")
    path = REPO_ROOT / "CHANGELOG.md"
    text = path.read_text()
    slug = repo_slug()
    titles = section_titles()

    lines = ["", "### Also shipped in this release", "",
             "<!-- Added by scripts/check-release-changelog.py: these commits are",
             "     contained in the tag but were absent from the generated",
             "     changelog, because they merged after release-please last",
             "     regenerated the release PR. -->", ""]
    for sha, subject in missing:
        kind = CONVENTIONAL.match(subject).group("type")
        lines.append(f"{render_entry(sha, subject, slug)} — _{titles.get(kind, kind)}_")
    lines.append("")

    start = re.search(rf"^## \[{re.escape(version)}\]", text, re.MULTILINE)
    rest = text[start.end():]
    nxt = re.search(r"^## \[", rest, re.MULTILINE)
    cut = start.end() + (nxt.start() if nxt else len(rest))
    path.write_text(
        text[:cut].rstrip("\n") + "\n" + "\n".join(lines) + "\n" + text[cut:].lstrip("\n")
    )


def missing_entries(tag: str, prev: str | None, types: set[str]) -> list[tuple[str, str]]:
    span = f"{prev}..{tag}" if prev else tag
    log = run("git", "log", "--no-merges", "--format=%h%x00%s", span)
    section = changelog_section(tag.lstrip("v"))
    if section is None:
        # A tag with no section at all predates the changelog (or was cut by
        # hand). That is a different condition from a section that is missing
        # entries, and reporting it as drift would bury the real signal.
        raise NoSection(tag)

    missing = []
    for line in filter(None, log.splitlines()):
        sha, subject = line.split("\0", 1)
        if RELEASE_COMMIT.match(subject):
            continue
        match = CONVENTIONAL.match(subject)
        if match is None or match.group("type") not in types:
            continue
        # A PR number is how release-please identifies an entry; fall back to
        # the abbreviated sha, which it also writes, for a direct-push commit.
        pr = PR_REF.search(subject)
        needle = f"#{pr.group('pr')}" if pr else sha
        if needle not in section:
            missing.append((sha, subject))
    return missing


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--tag",
        help="release tag to check (default: the highest semver tag)",
    )
    parser.add_argument(
        "--fix",
        action="store_true",
        help="append the missing commits to this version's changelog section",
    )
    args = parser.parse_args()

    tag = args.tag or (sorted_tags() or [None])[-1]
    if tag is None:
        print("no semver tags yet; nothing to reconcile")
        return 0

    prev = previous_tag(tag)
    try:
        missing = missing_entries(tag, prev, changelog_types())
    except NoSection:
        print(f"{tag}: no changelog section; predates CHANGELOG.md, skipping")
        return 0

    span = f"{prev}..{tag}" if prev else tag
    if not missing:
        print(f"{tag}: changelog matches `git log {span}`")
        return 0

    print(f"{tag}: {len(missing)} commit(s) shipped but appear in no changelog entry")
    print(f"  range: {span}")
    for sha, subject in missing:
        print(f"  {sha}  {subject}")
    print()
    print()
    print("The code is released, so this is an accuracy defect in the release")
    print("record rather than a shipping defect.")

    if args.fix:
        apply_fix(tag, missing)
        print(f"CHANGELOG.md updated for {tag}. Commit it and update the release body.")
        return 0

    print("Re-run with --fix to append them to this version's section.")
    return 1


if __name__ == "__main__":
    sys.exit(main())
