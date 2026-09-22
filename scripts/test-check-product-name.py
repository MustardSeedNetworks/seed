#!/usr/bin/env python3
"""Self-tests for the product-name gate."""

from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path

CHECKER = Path(__file__).with_name("check-product-name.py")


class ProductNameTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def write(self, path: str, content: str) -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")

    def run_checker(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["python3", str(CHECKER), "--root", str(self.root)],
            capture_output=True,
            text=True,
            check=False,
        )

    def test_clean_tree_passes(self) -> None:
        self.write("ui/src/app.tsx", "// Root component for Seed.\n")
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)

    def test_articled_name_fails(self) -> None:
        self.write("ui/src/app.tsx", "// Root component for The Seed.\n")
        result = self.run_checker()
        self.assertEqual(1, result.returncode)
        self.assertIn("ui/src/app.tsx:1", result.stdout)

    def test_upper_case_banner_fails(self) -> None:
        self.write("cmd/seed/root.go", 'x := "THE SEED - SETUP"\n')
        result = self.run_checker()
        self.assertEqual(1, result.returncode)
        self.assertIn("cmd/seed/root.go:1", result.stdout)

    def test_lower_case_article_is_ordinary_prose(self) -> None:
        # The reason this gate is case-sensitive: stem's UI-STEM-8 tried the
        # case-folding banned-vocabulary matcher and reverted it over exactly
        # this sentence shape.
        self.write("cmd/seed/root.go", "// Restart the seed daemon to apply.\n")
        self.write("ui/src/api/client.ts", "// talks to the Seed backend\n")
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)

    def test_out_of_scope_paths_are_not_scanned(self) -> None:
        # The identifiers UI-SEED-9 deliberately left alone, and the history
        # that must not be rewritten, all live outside SCOPE.
        self.write("internal/auth/token.go", 'Issuer: "The Seed",\n')
        self.write("CHANGELOG.md", "The Seed 0.1.0\n")
        self.write("docs/ARCHITECTURE.md", "# The Seed - System Architecture\n")
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)

    def test_build_output_is_skipped(self) -> None:
        self.write("ui/src/node_modules/pkg/index.js", 'var x = "The Seed";\n')
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
