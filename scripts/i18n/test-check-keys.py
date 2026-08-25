#!/usr/bin/env python3
"""Self-tests for the i18n key cross-reference gate.

The gate reported findings as `::error::` annotations while returning 0
whenever there were 30 or fewer of them, because `code = 1` sat inside the
`if len(...) > 30:` branch that prints the "… and N more" line. A GitHub
Actions annotation does not fail a job, so a repo could report unreferenced
keys on every run and stay green. Nothing tested the exit code, so nothing
caught it.

check-keys.py derives its ROOT from __file__ and reads the allowlist relative
to the working directory, so each case builds a throwaway repo tree with the
checker copied to the same relative path and runs it there.
"""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

CHECKER = Path(__file__).with_name("check-keys.py")


class CheckKeysTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        (self.root / "scripts" / "i18n").mkdir(parents=True)
        shutil.copy(CHECKER, self.root / "scripts" / "i18n" / "check-keys.py")
        (self.root / "ui" / "src").mkdir(parents=True)
        (self.root / "internal" / "i18n" / "locales" / "en").mkdir(parents=True)

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def locale(self, namespace: str, payload: dict) -> None:
        path = self.root / "internal" / "i18n" / "locales" / "en" / f"{namespace}.json"
        path.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    def source(self, name: str, content: str) -> None:
        (self.root / "ui" / "src" / name).write_text(content, encoding="utf-8")

    def run_checker(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, "scripts/i18n/check-keys.py"],
            cwd=self.root,
            capture_output=True,
            text=True,
            check=False,
        )

    def test_clean_tree_passes(self) -> None:
        self.locale("common", {"greeting": {"hello": "Hello"}})
        self.source(
            "Greeting.tsx",
            "const { t } = useTranslation('common');\n"
            "export const G = () => <p>{t('greeting.hello')}</p>;\n",
        )
        self.assertEqual(self.run_checker().returncode, 0)

    def test_single_unused_key_fails(self) -> None:
        """One unreferenced key must fail. This is the regression: it exited 0."""
        self.locale(
            "common",
            {"greeting": {"hello": "Hello", "abandoned": "Nothing references me"}},
        )
        self.source(
            "Greeting.tsx",
            "const { t } = useTranslation('common');\n"
            "export const G = () => <p>{t('greeting.hello')}</p>;\n",
        )
        result = self.run_checker()
        self.assertIn("greeting.abandoned", result.stdout)
        self.assertEqual(result.returncode, 1, "one unused key must fail the gate")

    def test_single_missing_key_fails(self) -> None:
        """One t() call with no locale entry must fail, for the same reason."""
        self.locale("common", {"greeting": {"hello": "Hello"}})
        self.source(
            "Greeting.tsx",
            "const { t } = useTranslation('common');\n"
            "export const G = () => <p>{t('greeting.absent')}</p>;\n",
        )
        result = self.run_checker()
        self.assertIn("greeting.absent", result.stdout)
        self.assertEqual(result.returncode, 1, "one missing key must fail the gate")

    def test_trans_i18nkey_counts_as_a_reference(self) -> None:
        """<Trans i18nKey> is a reference; it used to read as an unused key."""
        self.locale("common", {"greeting": {"rich": "Hello <strong>you</strong>"}})
        self.source(
            "Greeting.tsx",
            "const { t } = useTranslation('common');\n"
            'export const G = () => <Trans i18nKey="greeting.rich" '
            "components={{ strong: <strong /> }} />;\n",
        )
        result = self.run_checker()
        self.assertEqual(
            result.returncode, 0, f"<Trans> key reported unused:\n{result.stdout}"
        )

    def test_trans_i18nkey_missing_from_locale_fails(self) -> None:
        """A <Trans> pointing at a key that does not exist is now caught."""
        self.locale("common", {"greeting": {"rich": "Hello"}})
        self.source(
            "Greeting.tsx",
            "const { t } = useTranslation('common');\n"
            'export const G = () => <Trans i18nKey="greeting.typo" />;\n',
        )
        result = self.run_checker()
        self.assertEqual(result.returncode, 1)
        self.assertIn("greeting.typo", result.stdout)


if __name__ == "__main__":
    unittest.main()
