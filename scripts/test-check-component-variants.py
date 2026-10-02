#!/usr/bin/env python3
"""Self-tests for the @layer components variant gate."""

from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path

CHECKER = Path(__file__).with_name("check-component-variants.py")

CSS = """\
@layer components {
  /* .commented { } is not a class */
  .pad { @apply p-4; background: url(grain.png); }
  .pad-lg,
  .stack > * + * { @apply p-6; }
  @media (min-width: 40rem) {
    .nested-rule { color: red; }
  }
}
@utility pad-utility { padding: 1rem; }
.outside { color: red; }
"""


class ComponentVariantsTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.write("ui/src/index.css", CSS)

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

    def assert_fails_on(self, markup: str, token: str) -> None:
        self.write("ui/src/ui/Shell.tsx", markup)
        result = self.run_checker()
        self.assertEqual(1, result.returncode, result.stdout)
        self.assertIn(f"ui/src/ui/Shell.tsx:1: {token}", result.stdout)

    def test_bare_component_classes_pass(self) -> None:
        self.write("ui/src/ui/Shell.tsx", '<div className="pad pad-lg stack" />\n')
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout)

    def test_responsive_variant_fails(self) -> None:
        self.assert_fails_on('<div className="pad sm:pad-lg" />\n', "sm:pad-lg")

    def test_state_variant_in_template_literal_fails(self) -> None:
        self.assert_fails_on("const c = `x focus:pad`;\n", "focus:pad")

    def test_stacked_and_important_variants_fail(self) -> None:
        self.assert_fails_on("cn('dark:lg:!stack')\n", "dark:lg:!stack")

    def test_every_selector_in_a_list_is_a_component_class(self) -> None:
        self.assert_fails_on('<p className="md:stack" />\n', "md:stack")

    def test_rule_nested_in_an_at_rule_is_a_component_class(self) -> None:
        self.assert_fails_on('<p className="sm:nested-rule" />\n', "sm:nested-rule")

    def test_variants_on_utilities_pass(self) -> None:
        self.write(
            "ui/src/ui/Shell.tsx",
            '<div className="sm:p-6 lg:pad-utility hover:outside" />\n',
        )
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout)

    def test_object_keys_urls_and_declaration_values_pass(self) -> None:
        self.write("ui/src/api.ts", "const x = { pad: 1 };\nfetch('https://pad');\nhover:png\n")
        result = self.run_checker()
        self.assertEqual(0, result.returncode, result.stdout)


if __name__ == "__main__":
    unittest.main()
