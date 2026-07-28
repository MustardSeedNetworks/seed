#!/usr/bin/env python3
"""Self-tests for the repository banned-vocabulary gate."""

from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path


CHECKER = Path(__file__).with_name("check-banned-vocabulary.py")


class BannedVocabularyTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        subprocess.run(["git", "init", "-q", str(self.root)], check=True)
        self.terms = self.root / "terms.txt"
        self.terms.write_text("AI\nAI-powered\nopen source\n", encoding="utf-8")
        self.exclusions = self.root / "exclusions.txt"
        self.exclusions.write_text(
            "CHANGELOG.md | immutable release history\n"
            "internal/database/migrations/** | immutable migrations\n"
            "**/AGENTS.md | developer instructions\n",
            encoding="utf-8",
        )

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def write(self, path: str, content: str) -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        subprocess.run(["git", "-C", str(self.root), "add", "-f", path], check=True)

    def run_checker(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                "python3",
                str(CHECKER),
                "--root",
                str(self.root),
                "--terms",
                str(self.terms),
                "--exclusions",
                str(self.exclusions),
            ],
            check=False,
            capture_output=True,
            text=True,
        )

    def test_positive_fixture_fails_with_location(self) -> None:
        self.write("README.md", "An AI-powered network tool.\n")
        result = self.run_checker()
        self.assertEqual(1, result.returncode)
        self.assertIn("README.md:1: banned term 'AI-powered'", result.stdout)

    def test_negative_fixture_passes(self) -> None:
        self.write("README.md", "A source-available network tool.\n")
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)

    def test_matching_is_case_insensitive(self) -> None:
        self.write("docs/current.md", "This is OPEN SOURCE.\n")
        result = self.run_checker()
        self.assertEqual(1, result.returncode)
        self.assertIn("docs/current.md:1: banned term 'open source'", result.stdout)

    def test_only_documented_paths_are_excluded(self) -> None:
        self.write("CHANGELOG.md", "AI-powered\n")
        self.write("internal/database/migrations/00001.sql", "AI-powered\n")
        self.write("dev/AGENTS.md", "AI-powered\n")
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
