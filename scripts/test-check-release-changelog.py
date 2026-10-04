#!/usr/bin/env python3
"""Self-tests for the release changelog gate.

Each case builds a throwaway repository holding a copy of the checker, which
resolves the repository from its own location.
"""

from __future__ import annotations

import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

CHECKER = Path(__file__).with_name("check-release-changelog.py")

CONFIG = {
    "changelog-sections": [
        {"type": "feat", "section": "Features"},
        {"type": "fix", "section": "Bug Fixes"},
        {"type": "chore", "section": "Miscellaneous"},
    ]
}

FIXED = "## [0.2.0](https://example.invalid) (2026-01-01)\n\n### Bug Fixes\n\n* defect (#2)\n"


class Repo:
    def __init__(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        (self.root / "scripts").mkdir()
        shutil.copy(CHECKER, self.root / "scripts" / CHECKER.name)
        (self.root / ".github").mkdir()
        (self.root / ".github" / "release-please-config.json").write_text(json.dumps(CONFIG))
        (self.root / "CHANGELOG.md").write_text("# Changelog\n")
        self.git("init", "--quiet", "--initial-branch=main")
        self.git("config", "user.email", "gate@example.invalid")
        self.git("config", "user.name", "Gate Test")
        self.commit("chore: start the tree (#1)")
        self.git("tag", "v0.1.0")

    def git(self, *args: str) -> None:
        subprocess.run(("git", *args), cwd=self.root, check=True, capture_output=True)

    def commit(self, subject: str) -> None:
        (self.root / "tree.txt").write_text(subject)
        self.git("add", "-A")
        self.git("commit", "--quiet", "-m", subject)

    def release(self, section: str, tag: bool) -> None:
        """release-please's commit: the new section on top of the changelog."""
        (self.root / "CHANGELOG.md").write_text("# Changelog\n\n" + section)
        self.git("add", "-A")
        self.git("commit", "--quiet", "-m", "chore(main): release 0.2.0 (#10)")
        if tag:
            self.git("tag", "v0.2.0")

    def check(self, *args: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["python3", str(self.root / "scripts" / CHECKER.name), *args],
            cwd=self.root,
            capture_output=True,
            text=True,
            check=False,
        )


class ReleaseChangelogTest(unittest.TestCase):
    def setUp(self) -> None:
        self.repo = Repo()
        self.addCleanup(self.repo.temp_dir.cleanup)

    def assertResult(self, result: subprocess.CompletedProcess[str], code: int) -> None:
        self.assertEqual(code, result.returncode, result.stdout + result.stderr)

    def test_tag_with_every_commit_passes(self) -> None:
        self.repo.commit("fix: defect (#2)")
        self.repo.release(FIXED, tag=True)
        self.assertResult(self.repo.check(), 0)

    def test_tag_missing_a_commit_fails(self) -> None:
        self.repo.commit("fix: defect (#2)")
        self.repo.commit("feat: shipped unrecorded (#3)")
        self.repo.release(FIXED, tag=True)
        result = self.repo.check("--tag", "v0.2.0")
        self.assertResult(result, 1)
        self.assertIn("(#3)", result.stdout)

    def test_unrendered_type_is_not_required(self) -> None:
        self.repo.commit("fix: defect (#2)")
        self.repo.commit("build: repackage (#3)")
        self.repo.release(FIXED, tag=True)
        self.assertResult(self.repo.check(), 0)

    def test_pending_release_that_describes_the_tree_passes(self) -> None:
        self.repo.commit("fix: defect (#2)")
        self.repo.release(FIXED, tag=False)
        self.assertResult(self.repo.check("--pending"), 0)

    def test_pending_release_missing_a_pr_queued_ahead_fails(self) -> None:
        # #2769: a PR merged after release-please last regenerated the queued
        # release PR, so the release commit sits on top of it.
        self.repo.commit("fix: defect (#2)")
        self.repo.commit("feat: queued ahead of the release (#4)")
        self.repo.release(FIXED, tag=False)
        result = self.repo.check("--pending")
        self.assertResult(result, 1)
        self.assertIn("(#4)", result.stdout)

    def test_pending_release_ignores_a_pr_queued_behind(self) -> None:
        # The tag lands on the release commit, so this PR ships next release
        # and must not eject the release PR or itself.
        self.repo.commit("fix: defect (#2)")
        self.repo.release(FIXED, tag=False)
        self.repo.commit("feat: queued behind the release (#5)")
        self.assertResult(self.repo.check("--pending"), 0)

    def test_pending_after_the_tag_has_nothing_to_check(self) -> None:
        self.repo.commit("fix: defect (#2)")
        self.repo.release(FIXED, tag=True)
        self.repo.commit("feat: ordinary work after the release (#5)")
        self.assertResult(self.repo.check("--pending"), 0)

    def test_untagged_section_without_its_release_commit_fails(self) -> None:
        (self.repo.root / "CHANGELOG.md").write_text("# Changelog\n\n" + FIXED)
        self.repo.commit("fix: defect (#2)")
        self.assertResult(self.repo.check("--pending"), 1)


if __name__ == "__main__":
    unittest.main()
