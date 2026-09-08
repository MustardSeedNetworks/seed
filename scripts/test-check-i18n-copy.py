#!/usr/bin/env python3
"""Self-test for check-i18n-copy.py: builds a throwaway tree and proves the gate
goes red for each failure class, green when the tree is clean, and quiet on the
shapes that are not copy."""

from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("gate", HERE / "check-i18n-copy.py")
gate = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(gate)


class Tree:
    def __init__(self, source: str) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        src = self.root / "ui" / "src"
        src.mkdir(parents=True)
        (src / "Thing.tsx").write_text(source, encoding="utf-8")
        self.src = src

    def keys(self) -> list[str]:
        return sorted(key for key, _, _, _ in gate.sites(self.src, self.root))


def keys_for(source: str) -> list[str]:
    tree = Tree(source)
    try:
        return tree.keys()
    finally:
        tree.tmp.cleanup()


class FindsCopy(unittest.TestCase):
    def test_copy_prop_value(self) -> None:
        self.assertEqual(
            keys_for('<button aria-label="Refresh" />'),
            ["ui/src/Thing.tsx copy-prop aria-label=Refresh"],
        )

    def test_prop_value_of_any_length(self) -> None:
        # A single word is still read out by a screen reader.
        self.assertEqual(len(keys_for('<Input placeholder="Password" />')), 1)

    def test_string_in_a_jsx_expression(self) -> None:
        self.assertEqual(
            keys_for("<p>{loading ? 'Loading the rows' : rows.length}</p>"),
            ["ui/src/Thing.tsx literal Loading the rows"],
        )

    def test_string_in_a_label_map(self) -> None:
        self.assertEqual(
            keys_for("const L = { failed: 'Failed to load users' };"),
            ["ui/src/Thing.tsx literal Failed to load users"],
        )


class IgnoresWhatIsNotCopy(unittest.TestCase):
    def assert_clean(self, source: str) -> None:
        self.assertEqual(keys_for(source), [])

    def test_locale_key_argument(self) -> None:
        self.assert_clean("const x = t('Refresh the rows');")

    def test_prose_inside_a_comment(self) -> None:
        self.assert_clean("/* Renders the device list for this profile. */")

    def test_log_line(self) -> None:
        self.assert_clean("logger.warn('discovery', 'Failed to fetch service status');")

    def test_example_values(self) -> None:
        self.assert_clean(
            '<Input placeholder="192.168.1.1" />'
            '<Input placeholder="google.com" />'
            '<Input placeholder="rtsp://host:554/stream" />'
            '<Input placeholder="alice" />'
        )

    def test_code_token_that_reads_like_prose(self) -> None:
        # "Content-Type" and the tail of a ternary both look like two words to a
        # separator class that owns the hyphen or the quote.
        self.assert_clean("h['Content-Type'] = 'application/json';\nconst m = 'Hide';")

    def test_class_names_and_ids(self) -> None:
        self.assert_clean('<div className="flex items-center" data-testid="thing-row" />')

    def test_test_and_story_files_are_out_of_scope(self) -> None:
        tree = Tree("")
        try:
            (tree.src / "Thing.stories.tsx").write_text('<b aria-label="Refresh" />')
            (tree.src / "Thing.test.tsx").write_text('<b aria-label="Refresh" />')
            self.assertEqual(tree.keys(), [])
        finally:
            tree.tmp.cleanup()


class Baseline(unittest.TestCase):
    def test_stale_entry_is_reported(self) -> None:
        path = Path(tempfile.mkdtemp()) / "baseline.txt"
        path.write_text("# a reason\nui/src/Gone.tsx literal Was here once\n")
        self.assertEqual(
            gate.load_baseline(path),
            {"ui/src/Gone.tsx literal Was here once": "ui/src/Gone.tsx literal Was here once"},
        )

    def test_repo_baseline_is_clean(self) -> None:
        self.assertEqual(gate.main(), 0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
